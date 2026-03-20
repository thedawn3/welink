import { render, screen, waitFor } from '@testing-library/react';
import { MonthMessagesPanel } from './SentimentChart';
import { contactsApi } from '../../services/api';

vi.mock('../../services/api', () => ({
  contactsApi: {
    getMonthMessages: vi.fn(),
  },
}));

const mockedContactsApi = vi.mocked(contactsApi);

describe('MonthMessagesPanel', () => {
  it('reloads the open month when refreshKey changes', async () => {
    mockedContactsApi.getMonthMessages
      .mockResolvedValueOnce([{ time: '09:00', content: 'march old', is_mine: false, type: 1 }])
      .mockResolvedValueOnce([{ time: '09:05', content: 'march new', is_mine: false, type: 1 }]);

    const { rerender } = render(
      <MonthMessagesPanel
        username="alice"
        month="2024-03"
        contactName="Alice"
        includeMine={true}
        refreshKey={1}
        onClose={() => {}}
      />
    );

    expect(await screen.findByText('march old')).toBeInTheDocument();
    expect(mockedContactsApi.getMonthMessages).toHaveBeenCalledTimes(1);

    rerender(
      <MonthMessagesPanel
        username="alice"
        month="2024-03"
        contactName="Alice"
        includeMine={true}
        refreshKey={2}
        onClose={() => {}}
      />
    );

    await waitFor(() => {
      expect(mockedContactsApi.getMonthMessages).toHaveBeenCalledTimes(2);
    });
    expect(await screen.findByText('march new')).toBeInTheDocument();
  });
});
