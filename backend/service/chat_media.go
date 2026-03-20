package service

import (
	"bytes"
	"crypto/aes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"unicode"
	"welink/backend/config"
	"welink/backend/pkg/db"
)

var (
	errChatImageNotFound   = errors.New("chat image not found")
	errChatImageNoMsgDir   = errors.New("message media directory not configured")
	errChatImageNeedsAES   = errors.New("image aes key required for v2 image")
	errChatImageBadRequest = errors.New("invalid chat image request")
)

var (
	ServiceErrImageNotFound = errChatImageNotFound
	ServiceErrImageNeedsAES = errChatImageNeedsAES
)

var (
	imageMagicJPG  = []byte{0xFF, 0xD8, 0xFF}
	imageMagicPNG  = []byte{0x89, 0x50, 0x4E, 0x47}
	imageMagicGIF  = []byte("GIF")
	imageMagicBMP  = []byte("BM")
	imageMagicRIFF = []byte("RIFF")
	imageMagicWEBP = []byte("WEBP")
	imageMagicTIFF = []byte{0x49, 0x49, 0x2A, 0x00}
	v1Magic        = []byte{0x07, 0x08, 'V', '1', 0x08, 0x07}
	v2Magic        = []byte{0x07, 0x08, 'V', '2', 0x08, 0x07}
)

type chatMediaResolver struct {
	msgDir            string
	cacheDir          string
	imageAESKey       string
	imageAESKeySource string
	imageKeyMap       map[string]string
	imageKeyMode      string
	imageKeyCount     int
	wechatDecryptDir  string
	xorKey            byte
	mu                sync.Mutex
}

type chatImageAsset struct {
	MD5       string
	ThumbURL  string
	MediaURL  string
	Status    string
	available bool
}

type ChatMediaConfigStatus struct {
	MsgDir             string   `json:"msg_dir,omitempty"`
	WechatDecryptDir   string   `json:"wechat_decrypt_dir,omitempty"`
	MsgDirExists       bool     `json:"msg_dir_exists"`
	ImagePreviewReady  bool     `json:"image_preview_ready"`
	V2Detected         bool     `json:"v2_detected"`
	ImageAESKeyPresent bool     `json:"image_aes_key_present"`
	ImageAESKeySource  string   `json:"image_aes_key_source,omitempty"`
	ImageKeyMode       string   `json:"image_key_mode,omitempty"`
	ImageKeyCount      int      `json:"image_key_count,omitempty"`
	Issues             []string `json:"issues,omitempty"`
	Warnings           []string `json:"warnings,omitempty"`
	SuggestedCommand   string   `json:"suggested_command,omitempty"`
}

type chatImageKeyMaterial struct {
	SingleKey        string
	SingleKeySource  string
	KeyMap           map[string]string
	KeyMapSource     string
	KeyMode          string
	KeyCount         int
	WechatDecryptDir string
	SuggestedCommand string
}

func newChatMediaResolver(cfg *config.Config) *chatMediaResolver {
	msgDir := strings.TrimSpace(cfg.Data.MsgDir)
	if msgDir == "" {
		return nil
	}

	cacheRoot := strings.TrimSpace(cfg.Ingest.WorkDir)
	if cacheRoot == "" {
		cacheRoot = "./workdir"
	}

	keyMaterial := resolveChatImageKeyMaterial()

	return &chatMediaResolver{
		msgDir:            msgDir,
		cacheDir:          filepath.Join(cacheRoot, "chat-media-cache"),
		imageAESKey:       keyMaterial.SingleKey,
		imageAESKeySource: keyMaterial.SingleKeySource,
		imageKeyMap:       keyMaterial.KeyMap,
		imageKeyMode:      keyMaterial.KeyMode,
		imageKeyCount:     keyMaterial.KeyCount,
		wechatDecryptDir:  keyMaterial.WechatDecryptDir,
		xorKey:            0x88,
	}
}

func (s *ContactService) ResolveChatImage(username string, ts int64, md5Value, size string) (string, string, error) {
	if s == nil || s.mediaResolver == nil {
		return "", "", errChatImageNoMsgDir
	}
	return s.mediaResolver.resolveImageFile(username, ts, md5Value, size)
}

func (r *chatMediaResolver) buildChatImageAsset(username string, ts int64, packedInfo []byte) chatImageAsset {
	if r == nil || strings.TrimSpace(username) == "" || ts <= 0 {
		return chatImageAsset{}
	}

	md5Value := extractPackedInfoMD5(packedInfo)
	if md5Value == "" {
		return chatImageAsset{Status: "missing_md5"}
	}

	thumbSource, thumbEnc := r.findImageSource(username, ts, md5Value, "thumb")
	fullSource, fullEnc := r.findImageSource(username, ts, md5Value, "full")
	if thumbSource == "" && fullSource == "" {
		return chatImageAsset{MD5: md5Value, Status: "missing_source"}
	}

	if thumbSource == "" {
		thumbSource = fullSource
		thumbEnc = fullEnc
	}
	if fullSource == "" {
		fullSource = thumbSource
		fullEnc = thumbEnc
	}

	thumbReady := thumbSource != "" && r.canResolveImageSource(thumbSource, thumbEnc)
	fullReady := fullSource != "" && r.canResolveImageSource(fullSource, fullEnc)
	if !thumbReady && !fullReady {
		return chatImageAsset{MD5: md5Value, Status: "missing_aes_key"}
	}

	thumbURL := ""
	if thumbReady {
		thumbURL = r.buildImageURL(username, ts, md5Value, "thumb")
	} else if fullReady {
		thumbURL = r.buildImageURL(username, ts, md5Value, "full")
	}

	mediaURL := ""
	if fullReady {
		mediaURL = r.buildImageURL(username, ts, md5Value, "full")
	} else if thumbReady {
		mediaURL = r.buildImageURL(username, ts, md5Value, "thumb")
	}

	status := "ready"
	if !fullReady {
		status = "thumb_only"
	}

	return chatImageAsset{
		MD5:       md5Value,
		ThumbURL:  thumbURL,
		MediaURL:  mediaURL,
		Status:    status,
		available: true,
	}
}

func (r *chatMediaResolver) buildImageURL(username string, ts int64, md5Value, size string) string {
	values := url.Values{}
	values.Set("username", username)
	values.Set("ts", fmt.Sprintf("%d", ts))
	values.Set("md5", md5Value)
	values.Set("size", size)
	return "/api/media/chat-image?" + values.Encode()
}

func (r *chatMediaResolver) resolveImageFile(username string, ts int64, md5Value, size string) (string, string, error) {
	if r == nil || r.msgDir == "" {
		return "", "", errChatImageNoMsgDir
	}
	if strings.TrimSpace(username) == "" || strings.TrimSpace(md5Value) == "" || ts <= 0 {
		return "", "", errChatImageBadRequest
	}
	if !isHexMD5(md5Value) {
		return "", "", errChatImageBadRequest
	}

	sourcePath, _ := r.findImageSource(username, ts, md5Value, size)
	if sourcePath == "" {
		return "", "", errChatImageNotFound
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	cachePath, cacheMime, err := r.cachedDecodedPath(username, ts, md5Value, size)
	if err == nil && cachePath != "" {
		return cachePath, cacheMime, nil
	}

	payload, ext, err := r.decodeImageSource(sourcePath)
	if err != nil {
		return "", "", err
	}

	targetDir := filepath.Join(r.cacheDir, strings.TrimPrefix(db.GetTableName(username), "Msg_"), time.Unix(ts, 0).Format("2006-01"))
	if mkdirErr := os.MkdirAll(targetDir, 0o755); mkdirErr != nil {
		return "", "", mkdirErr
	}
	targetPath := filepath.Join(targetDir, fmt.Sprintf("%s_%s.%s", md5Value, size, ext))
	if writeErr := os.WriteFile(targetPath, payload, 0o644); writeErr != nil {
		return "", "", writeErr
	}
	return targetPath, mimeTypeByExt(ext, payload), nil
}

func (r *chatMediaResolver) cachedDecodedPath(username string, ts int64, md5Value, size string) (string, string, error) {
	baseDir := filepath.Join(r.cacheDir, strings.TrimPrefix(db.GetTableName(username), "Msg_"), time.Unix(ts, 0).Format("2006-01"))
	entries, err := os.ReadDir(baseDir)
	if err != nil {
		return "", "", err
	}
	prefix := md5Value + "_" + size + "."
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasPrefix(entry.Name(), prefix) {
			continue
		}
		fullPath := filepath.Join(baseDir, entry.Name())
		data, readErr := os.ReadFile(fullPath)
		if readErr != nil {
			continue
		}
		ext := strings.TrimPrefix(filepath.Ext(entry.Name()), ".")
		return fullPath, mimeTypeByExt(ext, data), nil
	}
	return "", "", os.ErrNotExist
}

func (r *chatMediaResolver) findImageSource(username string, ts int64, md5Value, size string) (string, bool) {
	tableHash := strings.TrimPrefix(db.GetTableName(username), "Msg_")
	if tableHash == "" {
		return "", false
	}
	months := imageMonthCandidates(ts)
	variants := imageVariantCandidates(size)
	for _, month := range months {
		baseDir := filepath.Join(r.msgDir, "attach", tableHash, month, "Img")
		for _, variant := range variants {
			fullPath := filepath.Join(baseDir, md5Value+variant)
			if info, err := os.Stat(fullPath); err == nil && !info.IsDir() {
				return fullPath, isEncryptedChatImage(fullPath)
			}
		}
	}
	return "", false
}

func imageMonthCandidates(ts int64) []string {
	t := time.Unix(ts, 0)
	curr := t.Format("2006-01")
	prev := t.AddDate(0, -1, 0).Format("2006-01")
	next := t.AddDate(0, 1, 0).Format("2006-01")
	result := []string{curr}
	if prev != curr {
		result = append(result, prev)
	}
	if next != curr && next != prev {
		result = append(result, next)
	}
	return result
}

func imageVariantCandidates(size string) []string {
	if size == "thumb" {
		return []string{"_t_M.dat", "_t.dat", "_M.dat", ".dat", "_h.dat"}
	}
	return []string{"_M.dat", ".dat", "_h.dat", "_t_M.dat", "_t.dat"}
}

func extractPackedInfoMD5(blob []byte) string {
	if len(blob) < 32 {
		return ""
	}
	for i := 0; i <= len(blob)-32; i++ {
		ch := blob[i]
		if !isHexByte(ch) {
			continue
		}
		ok := true
		for j := 0; j < 32; j++ {
			if !isHexByte(blob[i+j]) {
				ok = false
				break
			}
		}
		if ok {
			return strings.ToLower(string(blob[i : i+32]))
		}
	}
	return ""
}

func isHexByte(b byte) bool {
	return (b >= '0' && b <= '9') || (b >= 'a' && b <= 'f') || (b >= 'A' && b <= 'F')
}

func isHexMD5(value string) bool {
	if len(value) != 32 {
		return false
	}
	for i := 0; i < len(value); i++ {
		if !isHexByte(value[i]) {
			return false
		}
	}
	return true
}

func (r *chatMediaResolver) canDecodeWithoutAES(path string) bool {
	if strings.TrimSpace(path) == "" {
		return false
	}
	head, err := readFileHead(path, 16)
	if err != nil {
		return false
	}
	if looksLikeImage(head) {
		return true
	}
	if bytes.HasPrefix(head, v1Magic) {
		return true
	}
	if bytes.HasPrefix(head, v2Magic) {
		return false
	}
	return detectXORKey(head) >= 0
}

func isEncryptedChatImage(path string) bool {
	head, err := readFileHead(path, 16)
	if err != nil {
		return false
	}
	if bytes.HasPrefix(head, v1Magic) || bytes.HasPrefix(head, v2Magic) {
		return true
	}
	return !looksLikeImage(head)
}

func readFileHead(path string, size int) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if len(data) < size {
		return data, nil
	}
	return data[:size], nil
}

func (r *chatMediaResolver) decodeImageSource(path string) ([]byte, string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, "", err
	}

	if looksLikeImage(data) {
		return data, detectImageExt(data), nil
	}

	switch {
	case bytes.HasPrefix(data, v2Magic):
		if !r.hasAnyV2KeyMaterial() {
			return nil, "", errChatImageNeedsAES
		}
		payload, ext, decErr := r.decryptV2ImageAuto(data)
		if decErr != nil {
			return nil, "", decErr
		}
		return payload, ext, nil
	case bytes.HasPrefix(data, v1Magic):
		payload, ext, decErr := decryptV2Image(data, "cfcd208495d565ef", r.xorKey)
		if decErr != nil {
			return nil, "", decErr
		}
		return payload, ext, nil
	default:
		key := detectXORKey(data)
		if key < 0 {
			return nil, "", errChatImageNotFound
		}
		payload := xorDecrypt(data, byte(key))
		ext := detectImageExt(payload)
		return payload, ext, nil
	}
}

func (r *chatMediaResolver) hasAnyV2KeyMaterial() bool {
	return strings.TrimSpace(r.imageAESKey) != "" || len(r.imageKeyMap) > 0
}

func (r *chatMediaResolver) canResolveImageSource(path string, encrypted bool) bool {
	if strings.TrimSpace(path) == "" {
		return false
	}
	if !encrypted {
		return true
	}
	head, err := readFileHead(path, 31)
	if err != nil {
		return false
	}
	if looksLikeImage(head) {
		return true
	}
	if bytes.HasPrefix(head, v1Magic) {
		return true
	}
	if bytes.HasPrefix(head, v2Magic) {
		if strings.TrimSpace(r.imageAESKey) != "" {
			return true
		}
		if len(r.imageKeyMap) == 0 || len(head) < 31 {
			return false
		}
		ctHex := strings.ToLower(hex.EncodeToString(head[15:31]))
		return strings.TrimSpace(r.imageKeyMap[ctHex]) != ""
	}
	return r.canDecodeWithoutAES(path)
}

func (r *chatMediaResolver) decryptV2ImageAuto(data []byte) ([]byte, string, error) {
	candidates := r.candidateAESKeysForData(data)
	if len(candidates) == 0 {
		return nil, "", errChatImageNeedsAES
	}
	for _, candidate := range candidates {
		payload, ext, err := decryptV2Image(data, candidate, r.xorKey)
		if err == nil {
			return payload, ext, nil
		}
	}
	return nil, "", errChatImageNeedsAES
}

func (r *chatMediaResolver) candidateAESKeysForData(data []byte) []string {
	candidates := make([]string, 0, 4)
	seen := make(map[string]struct{})
	appendKey := func(key string) {
		if strings.TrimSpace(key) == "" {
			return
		}
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		candidates = append(candidates, key)
	}

	if len(data) >= 31 && len(r.imageKeyMap) > 0 {
		ctHex := strings.ToLower(hex.EncodeToString(data[15:31]))
		appendKey(r.imageKeyMap[ctHex])
	}

	appendKey(r.imageAESKey)

	if len(r.imageKeyMap) > 0 {
		for _, key := range r.imageKeyMap {
			appendKey(key)
		}
	}

	return candidates
}

func decryptV2Image(data []byte, aesKey string, xorKey byte) ([]byte, string, error) {
	if len(data) < 15 {
		return nil, "", errChatImageNotFound
	}
	keyBytes := []byte(aesKey)
	if len(keyBytes) < 16 {
		return nil, "", errChatImageNeedsAES
	}
	keyBytes = keyBytes[:16]

	aesSize := int(leUint32(data[6:10]))
	xorSize := int(leUint32(data[10:14]))
	if xorSize < 0 || aesSize < 0 || len(data) < 15 {
		return nil, "", errChatImageNotFound
	}

	alignedAESSize := aesSize
	if rem := alignedAESSize % aes.BlockSize; rem == 0 {
		alignedAESSize += aes.BlockSize
	} else {
		alignedAESSize += aes.BlockSize - rem
	}

	offset := 15
	if offset+alignedAESSize > len(data) || xorSize > len(data) {
		return nil, "", errChatImageNotFound
	}

	block, err := aes.NewCipher(keyBytes)
	if err != nil {
		return nil, "", err
	}
	decAES := make([]byte, alignedAESSize)
	for i := 0; i < alignedAESSize; i += aes.BlockSize {
		block.Decrypt(decAES[i:i+aes.BlockSize], data[offset+i:offset+i+aes.BlockSize])
	}
	if aesSize > len(decAES) {
		return nil, "", errChatImageNotFound
	}
	decAES = decAES[:aesSize]
	offset += alignedAESSize
	rawEnd := len(data) - xorSize
	if rawEnd < offset {
		rawEnd = offset
	}
	rawData := data[offset:rawEnd]
	xorData := xorDecrypt(data[rawEnd:], xorKey)

	payload := make([]byte, 0, len(decAES)+len(rawData)+len(xorData))
	payload = append(payload, decAES...)
	payload = append(payload, rawData...)
	payload = append(payload, xorData...)
	return payload, detectImageExt(payload), nil
}

func xorDecrypt(data []byte, key byte) []byte {
	out := make([]byte, len(data))
	for i, b := range data {
		out[i] = b ^ key
	}
	return out
}

func detectXORKey(data []byte) int {
	if len(data) < 4 {
		return -1
	}
	magics := [][]byte{
		imageMagicPNG,
		imageMagicGIF,
		imageMagicTIFF,
		imageMagicRIFF,
		imageMagicJPG,
	}
	for _, magic := range magics {
		key := int(data[0] ^ magic[0])
		match := true
		for i := 1; i < len(magic) && i < len(data); i++ {
			if data[i]^byte(key) != magic[i] {
				match = false
				break
			}
		}
		if match {
			return key
		}
	}
	if len(data) >= 2 {
		key := int(data[0] ^ imageMagicBMP[0])
		if data[1]^byte(key) == imageMagicBMP[1] {
			return key
		}
	}
	return -1
}

func detectImageExt(data []byte) string {
	switch {
	case len(data) >= 3 && bytes.Equal(data[:3], imageMagicJPG):
		return "jpg"
	case len(data) >= 4 && bytes.Equal(data[:4], imageMagicPNG):
		return "png"
	case len(data) >= 3 && bytes.Equal(data[:3], imageMagicGIF):
		return "gif"
	case len(data) >= 2 && bytes.Equal(data[:2], imageMagicBMP):
		return "bmp"
	case len(data) >= 12 && bytes.Equal(data[:4], imageMagicRIFF) && bytes.Equal(data[8:12], imageMagicWEBP):
		return "webp"
	case len(data) >= 4 && bytes.Equal(data[:4], imageMagicTIFF):
		return "tif"
	default:
		return "bin"
	}
}

func looksLikeImage(data []byte) bool {
	return detectImageExt(data) != "bin"
}

func mimeTypeByExt(ext string, payload []byte) string {
	switch strings.ToLower(strings.TrimPrefix(ext, ".")) {
	case "jpg", "jpeg":
		return "image/jpeg"
	case "png":
		return "image/png"
	case "gif":
		return "image/gif"
	case "bmp":
		return "image/bmp"
	case "webp":
		return "image/webp"
	case "tif", "tiff":
		return "image/tiff"
	default:
		if len(payload) > 0 {
			return http.DetectContentType(payload)
		}
		return "application/octet-stream"
	}
}

func leUint32(data []byte) uint32 {
	if len(data) < 4 {
		return 0
	}
	return uint32(data[0]) | uint32(data[1])<<8 | uint32(data[2])<<16 | uint32(data[3])<<24
}

func safeMessageID(parts ...string) string {
	filtered := make([]string, 0, len(parts))
	for _, part := range parts {
		if strings.TrimSpace(part) == "" {
			continue
		}
		sb := strings.Builder{}
		for _, r := range part {
			switch {
			case unicode.IsLetter(r), unicode.IsDigit(r):
				sb.WriteRune(r)
			case r == '-', r == '_', r == ':':
				sb.WriteRune(r)
			default:
				sb.WriteRune('_')
			}
		}
		filtered = append(filtered, sb.String())
	}
	return strings.Join(filtered, ":")
}

func normalizeImageAESKey(value string) string {
	value = strings.TrimSpace(value)
	if len(value) == 32 && isHexMD5(strings.ToLower(value)) {
		if decoded, err := hex.DecodeString(value); err == nil && len(decoded) == 16 {
			return string(decoded)
		}
	}
	return value
}

func resolveChatImageKeyMaterial() chatImageKeyMaterial {
	material := chatImageKeyMaterial{
		KeyMap: make(map[string]string),
	}

	displayDir := strings.TrimSpace(os.Getenv("WELINK_WECHAT_DECRYPT_DIR"))
	accessDir := strings.TrimSpace(os.Getenv("WELINK_WECHAT_DECRYPT_MOUNT_DIR"))
	if accessDir == "" {
		accessDir = displayDir
	}
	material.WechatDecryptDir = displayDir
	if material.WechatDecryptDir == "" {
		material.WechatDecryptDir = accessDir
	}

	if value := normalizeImageAESKey(strings.TrimSpace(os.Getenv("WELINK_IMAGE_AES_KEY"))); value != "" {
		material.SingleKey = value
		material.SingleKeySource = "env:WELINK_IMAGE_AES_KEY"
	}

	if filePath := strings.TrimSpace(os.Getenv("WELINK_IMAGE_AES_KEY_FILE")); filePath != "" {
		if singleKey, keyMap := readImageKeyMaterialFile(filePath); material.SingleKey == "" && singleKey != "" {
			material.SingleKey = singleKey
			material.SingleKeySource = "file:" + filePath
		} else if len(keyMap) > 0 {
			material.KeyMap = keyMap
			material.KeyMapSource = "file:" + filePath
		}
	}

	if filePath := strings.TrimSpace(os.Getenv("WELINK_WECHAT_DECRYPT_CONFIG")); filePath != "" && material.SingleKey == "" && len(material.KeyMap) == 0 {
		if singleKey, keyMap := readImageKeyMaterialFile(filePath); singleKey != "" {
			material.SingleKey = singleKey
			material.SingleKeySource = "file:" + filePath
		} else if len(keyMap) > 0 {
			material.KeyMap = keyMap
			material.KeyMapSource = "file:" + filePath
		}
		if material.WechatDecryptDir == "" {
			material.WechatDecryptDir = filepath.Dir(filePath)
		}
	}

	if accessDir != "" {
		mapPath := filepath.Join(accessDir, "image_keys.json")
		if len(material.KeyMap) == 0 {
			if _, keyMap := readImageKeyMaterialFile(mapPath); len(keyMap) > 0 {
				material.KeyMap = keyMap
				material.KeyMapSource = "file:" + mapPath
			}
		}

		configPath := filepath.Join(accessDir, "config.json")
		if material.SingleKey == "" {
			if singleKey, _ := readImageKeyMaterialFile(configPath); singleKey != "" {
				material.SingleKey = singleKey
				material.SingleKeySource = "file:" + configPath
			}
		}
	}

	if len(material.KeyMap) > 0 {
		material.KeyMode = "map"
		material.KeyCount = len(material.KeyMap)
	} else if material.SingleKey != "" {
		material.KeyMode = "single"
		material.KeyCount = 1
	}

	if material.WechatDecryptDir != "" {
		material.SuggestedCommand = "cd " + material.WechatDecryptDir + " && sudo ./find_image_key"
	} else {
		material.SuggestedCommand = "cd /path/to/wechat-decrypt && sudo ./find_image_key"
	}

	return material
}

func readImageKeyMaterialFile(path string) (string, map[string]string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", nil
	}
	trimmed := strings.TrimSpace(string(data))
	if trimmed == "" {
		return "", nil
	}

	if strings.HasPrefix(trimmed, "{") {
		var payload map[string]any
		if err := json.Unmarshal(data, &payload); err == nil {
			if value, ok := payload["image_aes_key"].(string); ok {
				return normalizeImageAESKey(value), nil
			}
			if keyMap := parseImageAESKeyMap(payload); len(keyMap) > 0 {
				return "", keyMap
			}
			return "", nil
		}
	}

	return normalizeImageAESKey(trimmed), nil
}

func parseImageAESKeyMap(payload map[string]any) map[string]string {
	keyMap := make(map[string]string)
	for ctHex, rawValue := range payload {
		value, ok := rawValue.(string)
		if !ok {
			continue
		}
		if !isHexMD5(strings.ToLower(strings.TrimSpace(ctHex))) {
			continue
		}
		normalized := normalizeImageAESKey(value)
		if len(normalized) < 16 {
			continue
		}
		keyMap[strings.ToLower(strings.TrimSpace(ctHex))] = normalized
	}
	if len(keyMap) == 0 {
		return nil
	}
	return keyMap
}

func InspectChatMediaConfig(cfg *config.Config) ChatMediaConfigStatus {
	status := ChatMediaConfigStatus{
		MsgDir: strings.TrimSpace(cfg.Data.MsgDir),
	}
	keyMaterial := resolveChatImageKeyMaterial()
	status.WechatDecryptDir = keyMaterial.WechatDecryptDir
	if status.MsgDir == "" {
		status.Warnings = append(status.Warnings, "未配置 WELINK_MSG_DIR，聊天图片只能显示为占位")
		if keyMaterial.WechatDecryptDir == "" {
			status.Warnings = append(status.Warnings, "如需自动读取 wechat-decrypt 的图片密钥结果，可配置 WELINK_WECHAT_DECRYPT_DIR")
		}
		return status
	}

	info, err := os.Stat(status.MsgDir)
	if err != nil || !info.IsDir() {
		status.Issues = append(status.Issues, "媒体目录不存在或不可访问")
		return status
	}
	status.MsgDirExists = true

	if keyMaterial.SingleKey != "" || len(keyMaterial.KeyMap) > 0 {
		status.ImageAESKeyPresent = true
		if keyMaterial.KeyMapSource != "" {
			status.ImageAESKeySource = keyMaterial.KeyMapSource
		} else {
			status.ImageAESKeySource = keyMaterial.SingleKeySource
		}
		status.ImageKeyMode = keyMaterial.KeyMode
		status.ImageKeyCount = keyMaterial.KeyCount
	}

	status.V2Detected = detectV2DatUnderMsgDir(status.MsgDir)
	status.ImagePreviewReady = true
	if status.V2Detected && !status.ImageAESKeyPresent {
		status.ImagePreviewReady = false
		status.Warnings = append(status.Warnings, "检测到 V2 图片文件，但当前未配置可用的图片密钥（可通过 WELINK_IMAGE_AES_KEY 或 WELINK_WECHAT_DECRYPT_DIR 自动加载）")
		status.SuggestedCommand = keyMaterial.SuggestedCommand
	}
	return status
}

func detectV2DatUnderMsgDir(msgDir string) bool {
	attachDir := filepath.Join(strings.TrimSpace(msgDir), "attach")
	checked := 0
	found := false
	_ = filepath.WalkDir(attachDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d == nil || d.IsDir() || found {
			return nil
		}
		if !strings.HasSuffix(strings.ToLower(d.Name()), ".dat") {
			return nil
		}
		checked++
		head, readErr := readFileHead(path, 6)
		if readErr == nil && bytes.HasPrefix(head, v2Magic) {
			found = true
			return filepath.SkipAll
		}
		if checked >= 200 {
			return filepath.SkipAll
		}
		return nil
	})
	return found
}
