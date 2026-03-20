package service

import (
	"bytes"
	"crypto/aes"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"welink/backend/pkg/db"
)

func TestExtractPackedInfoMD5(t *testing.T) {
	md5Value := "7646374bce20b92917907ffe273749fa"
	blob := append([]byte{0x08, 0x01, 0x10, 0x18}, []byte(md5Value)...)
	if got := extractPackedInfoMD5(blob); got != md5Value {
		t.Fatalf("expected %q, got %q", md5Value, got)
	}
}

func TestDetectXORKey(t *testing.T) {
	key := byte(0x12)
	encrypted := xorDecrypt([]byte{0xFF, 0xD8, 0xFF, 0xEE}, key)
	if got := detectXORKey(encrypted); got != int(key) {
		t.Fatalf("expected key %d, got %d", key, got)
	}
}

func TestDecryptV2Image(t *testing.T) {
	plain := []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46, 0x49, 0x46}
	key := []byte("1234567890abcdef")
	block, err := aes.NewCipher(key)
	if err != nil {
		t.Fatalf("new cipher failed: %v", err)
	}
	padded := pkcs7PadForTest(plain, aes.BlockSize)
	enc := make([]byte, len(padded))
	for i := 0; i < len(padded); i += aes.BlockSize {
		block.Encrypt(enc[i:i+aes.BlockSize], padded[i:i+aes.BlockSize])
	}

	data := bytes.NewBuffer(nil)
	data.Write(v2Magic)
	data.Write([]byte{byte(len(plain)), 0, 0, 0})
	data.Write([]byte{0, 0, 0, 0})
	data.WriteByte(0)
	data.Write(enc)

	got, ext, err := decryptV2Image(data.Bytes(), string(key), 0x88)
	if err != nil {
		t.Fatalf("decrypt v2 failed: %v", err)
	}
	if !bytes.Equal(got, plain) {
		t.Fatalf("unexpected decrypted payload: %x", got)
	}
	if ext != "jpg" {
		t.Fatalf("expected jpg, got %q", ext)
	}
}

func TestDecodeImageSourceWithImageKeyMap(t *testing.T) {
	plain := []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46, 0x49, 0x46}
	key := []byte("1234567890abcdef")
	block, err := aes.NewCipher(key)
	if err != nil {
		t.Fatalf("new cipher failed: %v", err)
	}
	padded := pkcs7PadForTest(plain, aes.BlockSize)
	enc := make([]byte, len(padded))
	for i := 0; i < len(padded); i += aes.BlockSize {
		block.Encrypt(enc[i:i+aes.BlockSize], padded[i:i+aes.BlockSize])
	}

	data := bytes.NewBuffer(nil)
	data.Write(v2Magic)
	data.Write([]byte{byte(len(plain)), 0, 0, 0})
	data.Write([]byte{0, 0, 0, 0})
	data.WriteByte(0)
	data.Write(enc)

	imagePath := filepath.Join(t.TempDir(), "sample.dat")
	if err := os.WriteFile(imagePath, data.Bytes(), 0o644); err != nil {
		t.Fatalf("write test image: %v", err)
	}

	ctHex := hex.EncodeToString(enc[:16])
	resolver := &chatMediaResolver{
		imageKeyMap:   map[string]string{ctHex: string(key)},
		imageKeyMode:  "map",
		imageKeyCount: 1,
		xorKey:        0x88,
	}

	got, ext, err := resolver.decodeImageSource(imagePath)
	if err != nil {
		t.Fatalf("decode image source failed: %v", err)
	}
	if !bytes.Equal(got, plain) {
		t.Fatalf("unexpected decoded payload: %x", got)
	}
	if ext != "jpg" {
		t.Fatalf("expected jpg, got %q", ext)
	}
}

func TestReadImageKeyMaterialFileIgnoresConfigWithoutImageKey(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(configPath, []byte(`{"db_dir":"/tmp/db","keys_file":"all_keys.json"}`), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	singleKey, keyMap := readImageKeyMaterialFile(configPath)
	if singleKey != "" {
		t.Fatalf("expected empty single key, got %q", singleKey)
	}
	if len(keyMap) != 0 {
		t.Fatalf("expected empty key map, got %#v", keyMap)
	}
}

func TestBuildChatImageAssetFallsBackToThumbWhenFullKeyMissing(t *testing.T) {
	msgDir := t.TempDir()
	username := "wxid_pk1hveu65djk22"
	tableHash := strings.TrimPrefix(db.GetTableName(username), "Msg_")
	baseDir := filepath.Join(msgDir, "attach", tableHash, "2026-03", "Img")
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	thumbCT := bytes.Repeat([]byte{0x11}, 16)
	fullCT := bytes.Repeat([]byte{0x22}, 16)
	thumbData := append(append(append([]byte{}, v2Magic...), make([]byte, 9)...), thumbCT...)
	fullData := append(append(append([]byte{}, v2Magic...), make([]byte, 9)...), fullCT...)
	md5Value := "1f5d3360db779f6fe39ca8cf901bb2b1"

	if err := os.WriteFile(filepath.Join(baseDir, md5Value+"_t.dat"), thumbData, 0o644); err != nil {
		t.Fatalf("write thumb: %v", err)
	}
	if err := os.WriteFile(filepath.Join(baseDir, md5Value+".dat"), fullData, 0o644); err != nil {
		t.Fatalf("write full: %v", err)
	}

	resolver := &chatMediaResolver{
		msgDir:      msgDir,
		imageKeyMap: map[string]string{hex.EncodeToString(thumbCT): "1234567890abcdef"},
	}
	asset := resolver.buildChatImageAsset(username, 1773333562, []byte(md5Value))
	if asset.Status != "thumb_only" {
		t.Fatalf("expected thumb_only status, got %+v", asset)
	}
	if asset.ThumbURL == "" || asset.MediaURL == "" {
		t.Fatalf("expected both urls, got %+v", asset)
	}
	if asset.ThumbURL != asset.MediaURL {
		t.Fatalf("expected media url fallback to thumb, got thumb=%s media=%s", asset.ThumbURL, asset.MediaURL)
	}
}

func pkcs7PadForTest(data []byte, blockSize int) []byte {
	pad := blockSize - (len(data) % blockSize)
	if pad == 0 {
		pad = blockSize
	}
	out := make([]byte, 0, len(data)+pad)
	out = append(out, data...)
	for i := 0; i < pad; i++ {
		out = append(out, byte(pad))
	}
	return out
}
