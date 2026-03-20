import { fireEvent, render, screen } from '@testing-library/react';
import { SystemRuntimeView } from './SystemRuntimeView';

describe('SystemRuntimeView', () => {
  it('renders merged runtime details and triggers actions', () => {
    const onRefresh = vi.fn();
    const onStartDecrypt = vi.fn();
    const onStopDecrypt = vi.fn();
    const onReindex = vi.fn();
    const onExportContact = vi.fn();
    const onExportGroup = vi.fn();
    const onExportSearch = vi.fn();

    render(
      <SystemRuntimeView
        backendStatus={{ is_indexing: false, is_initialized: true, total_cached: 12 }}
        runtime={{
          engine_type: 'windows',
          decrypt_state: 'ready',
          data_revision: 5,
          updated_at: '2026-03-20T04:00:00Z',
          last_message_at: '2026-03-19T23:00:00Z',
          last_sns_at: '2026-03-18T09:30:00Z',
        }}
        changes={{
          data_revision: 5,
          pending_changes: 0,
          last_change_reason: 'message_0.db,message_0.db-wal',
          items: [],
        }}
        tasks={[]}
        logs={[]}
        latestEvent={{ type: 'runtime.reindex.finished', message: 'reindex finished' }}
        eventsConnected={true}
        loading={false}
        error={null}
        actionNotice={null}
        meta={{ lastEventAt: '2026-03-20T04:01:00Z', lastRefreshAt: '2026-03-20T04:02:00Z', lastRefreshReason: 'sse' }}
        defaultContactUsername="alice_default"
        defaultGroupUsername="group_default@chatroom"
        defaultSearchQuery="晚安"
        defaultSearchIncludeMine={false}
        onRefresh={onRefresh}
        onStartDecrypt={onStartDecrypt}
        onStopDecrypt={onStopDecrypt}
        onReindex={onReindex}
        onExportContact={onExportContact}
        onExportGroup={onExportGroup}
        onExportSearch={onExportSearch}
      />
    );

    expect(screen.getByText('SSE 已连接')).toBeInTheDocument();
    expect(screen.getByText(/最近事件：/)).toBeInTheDocument();
    expect(screen.getByText(/最新变更原因：message_0.db,message_0.db-wal/)).toBeInTheDocument();
    expect(screen.getByText(/最近刷新：/)).toBeInTheDocument();
    expect(screen.getByText(/最近消息时间：/)).toBeInTheDocument();
    expect(screen.getByText(/最近朋友圈时间：/)).toBeInTheDocument();

    fireEvent.click(screen.getByRole('button', { name: /刷新状态/ }));
    fireEvent.click(screen.getByRole('button', { name: /启动解密/ }));
    expect(screen.getByRole('button', { name: /停止解密/ })).toBeDisabled();
    fireEvent.click(screen.getByRole('button', { name: /强制重建索引/ }));

    expect(onRefresh).toHaveBeenCalledTimes(1);
    expect(onStartDecrypt).toHaveBeenCalledWith(
      expect.objectContaining({ auto_refresh: true, wal_enabled: true })
    );
    expect(onStopDecrypt).not.toHaveBeenCalled();
    expect(onReindex).toHaveBeenCalledTimes(1);

    fireEvent.change(screen.getByPlaceholderText(/联系人 username/), { target: { value: 'alice' } });
    fireEvent.click(screen.getAllByRole('button', { name: /导出/ })[0]);
    expect(onExportContact).toHaveBeenCalledWith('alice', 200);

    fireEvent.click(screen.getAllByRole('button', { name: /导出/ })[2]);
    expect(onExportSearch).toHaveBeenCalledWith('晚安', false, 200);
  });

  it('disables start action and switches copy in docker manual mode', () => {
    const onStartDecrypt = vi.fn();
    const onStopDecrypt = vi.fn();

    render(
      <SystemRuntimeView
        backendStatus={{ is_indexing: false, is_initialized: true, total_cached: 1 }}
        runtime={{ engine_type: 'macos', decrypt_state: 'idle' }}
        configCheck={{
          deployment_target: 'docker',
          mode: 'manual-stage',
          can_start_sync: false,
          primary_issue: 'source 目录不是标准目录，必须包含 contact/contact.db 与 message/message_*.db',
          blocking_reasons: ['source 目录不是标准目录，必须包含 contact/contact.db 与 message/message_*.db'],
          source_dir: {
            path: '/app/source-data',
            exists: true,
            standard_layout: false,
            has_contact: false,
            has_message: false,
          },
          analysis_dir: { path: '/app/analysis-data', exists: true },
          decrypt: { supported: true },
        }}
        changes={{ data_revision: 1, pending_changes: 0, items: [] }}
        tasks={[]}
        logs={[]}
        latestEvent={null}
        eventsConnected={false}
        loading={false}
        error={null}
        actionNotice={null}
        onRefresh={vi.fn()}
        onStartDecrypt={onStartDecrypt}
        onStopDecrypt={onStopDecrypt}
        onReindex={vi.fn()}
        onExportContact={vi.fn()}
        onExportGroup={vi.fn()}
        onExportSearch={vi.fn()}
      />
    );

    const syncButton = screen.getByRole('button', { name: /校验并同步标准目录/ });
    const stopButton = screen.getByRole('button', { name: /停止解密/ });
    expect(syncButton).toBeDisabled();
    expect(screen.getByText(/当前不可启动同步/)).toBeInTheDocument();
    expect(screen.getByText(/source 目录不是标准目录/)).toBeInTheDocument();
    expect(screen.queryByText(/source_dir 与 analysis_dir 不能是同一目录/)).not.toBeInTheDocument();
    expect(screen.getByText(/Docker 手动同步模式/)).toBeInTheDocument();
    expect(stopButton).toBeDisabled();
    fireEvent.click(syncButton);
    fireEvent.click(stopButton);
    expect(onStartDecrypt).not.toHaveBeenCalled();
    expect(onStopDecrypt).not.toHaveBeenCalled();
  });

  it('shows neutral analysis-only hint instead of a blocking error', () => {
    render(
      <SystemRuntimeView
        backendStatus={{ is_indexing: false, is_initialized: true, total_cached: 1 }}
        runtime={{ deployment_target: 'docker', engine_type: 'welink', decrypt_state: 'idle' }}
        configCheck={{
          deployment_target: 'docker',
          mode: 'analysis-only',
          can_start_sync: false,
          analysis_dir: {
            path: '/app/analysis-data',
            exists: true,
            has_contact: true,
            has_message: true,
          },
          source_dir: {
            path: '',
            exists: false,
          },
          warnings: ['未配置 source_data_dir，当前只能分析已有 analysis 目录'],
        }}
        changes={{ data_revision: 1, pending_changes: 0, items: [] }}
        tasks={[]}
        logs={[]}
        latestEvent={null}
        eventsConnected={false}
        loading={false}
        error={null}
        actionNotice={null}
        onRefresh={vi.fn()}
        onStartDecrypt={vi.fn()}
        onStopDecrypt={vi.fn()}
        onReindex={vi.fn()}
        onExportContact={vi.fn()}
        onExportGroup={vi.fn()}
        onExportSearch={vi.fn()}
      />
    );

    expect(screen.getByText('当前处于只分析模式')).toBeInTheDocument();
    expect(screen.queryByText('当前不可启动同步')).not.toBeInTheDocument();
    expect(screen.getByRole('button', { name: /启动解密|校验并同步标准目录/ })).toBeDisabled();
  });
});
