import { render, screen, waitFor } from '@testing-library/react';
import { DayChatPanel } from './DayChatPanel';
import { contactsApi } from '../../services/api';

vi.mock('../../services/api', () => ({
  contactsApi: {
    getDayMessages: vi.fn(),
  },
}));

const mockedContactsApi = vi.mocked(contactsApi);

describe('DayChatPanel', () => {
  it('reloads the open day when refreshKey changes', async () => {
    mockedContactsApi.getDayMessages
      .mockResolvedValueOnce([{ time: '09:00', content: 'old message', is_mine: false, type: 1 }])
      .mockResolvedValueOnce([{ time: '09:01', content: 'new message', is_mine: false, type: 1 }]);

    const { rerender } = render(
      <DayChatPanel
        username="alice"
        date="2024-03-15"
        dayCount={1}
        contactName="Alice"
        refreshKey={1}
        onClose={() => {}}
      />
    );

    expect(await screen.findByText('old message')).toBeInTheDocument();
    expect(mockedContactsApi.getDayMessages).toHaveBeenCalledTimes(1);

    rerender(
      <DayChatPanel
        username="alice"
        date="2024-03-15"
        dayCount={1}
        contactName="Alice"
        refreshKey={2}
        onClose={() => {}}
      />
    );

    await waitFor(() => {
      expect(mockedContactsApi.getDayMessages).toHaveBeenCalledTimes(2);
    });
    expect(await screen.findByText('new message')).toBeInTheDocument();
  });
});
