import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { ContactModal } from './ContactModal';
import { contactsApi, relationsApi } from '../../services/api';

const fetchWordCloud = vi.fn();

vi.mock('../../hooks/useContacts', () => ({
  useWordCloud: () => ({
    data: [],
    loading: false,
    fetch: fetchWordCloud,
  }),
}));

vi.mock('../../services/api', () => ({
  contactsApi: {
    getDetail: vi.fn().mockResolvedValue({}),
    getSentiment: vi.fn().mockResolvedValue({ monthly: [], overall: 0.5, positive: 0, negative: 0, neutral: 0 }),
    getCommonGroups: vi.fn().mockResolvedValue([]),
    searchMessages: vi.fn(),
  },
  relationsApi: {
    getDetail: vi.fn().mockResolvedValue({ evidence_groups: [], controversial_labels: [] }),
    getControversyDetail: vi.fn().mockResolvedValue({ controversial_labels: [] }),
  },
}));

vi.mock('./WordCloudCanvas', () => ({ WordCloudCanvas: () => <div>WordCloudCanvas</div> }));
vi.mock('./ContactDetailCharts', () => ({ ContactDetailCharts: () => <div>ContactDetailCharts</div> }));
vi.mock('./SentimentChart', () => ({ SentimentChart: () => <div>SentimentChart</div> }));
vi.mock('./RelationInsightPanel', () => ({ RelationInsightPanel: () => <div>RelationInsightPanel</div> }));
vi.mock('./ControversyPanel', () => ({ ControversyPanel: () => <div>ControversyPanel</div> }));
vi.mock('./ContactTimelinePanel', () => ({ ContactTimelinePanel: () => <div>ContactTimelinePanel</div> }));

const mockedContactsApi = vi.mocked(contactsApi);
const mockedRelationsApi = vi.mocked(relationsApi);

describe('ContactModal', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockedContactsApi.getDetail.mockResolvedValue({});
    mockedContactsApi.getSentiment.mockResolvedValue({ monthly: [], overall: 0.5, positive: 0, negative: 0, neutral: 0 });
    mockedContactsApi.getCommonGroups.mockResolvedValue([]);
    mockedRelationsApi.getDetail.mockResolvedValue({ evidence_groups: [], controversial_labels: [] });
    mockedRelationsApi.getControversyDetail.mockResolvedValue({ controversial_labels: [] });
  });

  it('reruns current search tab query when refreshKey changes', async () => {
    mockedContactsApi.searchMessages
      .mockResolvedValueOnce([{ time: '10:00', date: '2024-03-15', content: 'first hit', is_mine: false, type: 1 }])
      .mockResolvedValueOnce([{ time: '10:01', date: '2024-03-15', content: 'second hit', is_mine: false, type: 1 }]);

    const contact = {
      username: 'alice',
      nickname: 'Alice',
      remark: '',
      alias: '',
      flag: 3,
      description: '',
      big_head_url: '',
      small_head_url: '',
      total_messages: 10,
      first_message_time: '2024-01-01 10:00',
      last_message_time: '2024-03-15 10:00',
      first_msg: 'hello',
    };

    const { rerender } = render(
      <ContactModal
        contact={contact}
        onClose={() => {}}
        initialTab="search"
        refreshKey={1}
      />
    );

    const input = await screen.findByPlaceholderText('搜索聊天内容...');
    fireEvent.change(input, { target: { value: 'hello' } });
    fireEvent.submit(input.closest('form') as HTMLFormElement);

    expect(await screen.findByText('first hit')).toBeInTheDocument();
    expect(mockedContactsApi.searchMessages).toHaveBeenCalledTimes(1);

    rerender(
      <ContactModal
        contact={contact}
        onClose={() => {}}
        initialTab="search"
        refreshKey={2}
      />
    );

    await waitFor(() => {
      expect(mockedContactsApi.searchMessages).toHaveBeenCalledTimes(2);
    });
    expect(await screen.findByText('second hit')).toBeInTheDocument();
    expect(screen.getByDisplayValue('hello')).toBeInTheDocument();
  });

  it('keeps current tab and search state when the same contact object is refreshed', async () => {
    const contact = {
      username: 'alice',
      nickname: 'Alice',
      remark: '',
      alias: '',
      flag: 3,
      description: '',
      big_head_url: '',
      small_head_url: '',
      total_messages: 10,
      first_message_time: '2024-01-01 10:00',
      last_message_time: '2024-03-15 10:00',
      first_msg: 'hello',
    };

    const { rerender } = render(
      <ContactModal
        contact={contact}
        onClose={() => {}}
        initialTab="detail"
        refreshKey={1}
      />
    );

    fireEvent.click(await screen.findByRole('button', { name: '搜索记录' }));
    const input = await screen.findByPlaceholderText('搜索聊天内容...');
    fireEvent.change(input, { target: { value: 'persist me' } });

    rerender(
      <ContactModal
        contact={{ ...contact, total_messages: 11 }}
        onClose={() => {}}
        initialTab="detail"
        refreshKey={1}
      />
    );

    expect(screen.getByPlaceholderText('搜索聊天内容...')).toHaveValue('persist me');
    expect(screen.getByRole('button', { name: '搜索记录' })).toHaveClass('text-[#07c160]');
    expect(fetchWordCloud).toHaveBeenCalledTimes(1);
  });
});
