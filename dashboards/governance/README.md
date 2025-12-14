# PAW Governance Portal

A comprehensive, production-ready governance portal for the PAW blockchain, enabling community participation in on-chain governance through proposal creation, voting, and analytics.

## Features

### Core Functionality
- **Proposal Management**
  - View all proposals with advanced filtering
  - Detailed proposal information with timeline visualization
  - Support for multiple proposal types (Text, Parameter Change, Software Upgrade, Community Spend)
  - Real-time proposal status tracking

- **Voting Interface**
  - Intuitive voting panel with four options (Yes, No, Abstain, No With Veto)
  - Vote tallying with interactive charts
  - Voting power calculation
  - Vote history tracking

- **Proposal Creation**
  - Multi-type proposal support
  - Form validation and preview
  - Deposit management
  - Guided workflow

- **Analytics Dashboard**
  - Proposal success rate visualization
  - Voting trends over time
  - Participation rate tracking
  - Top voter rankings

- **Governance Parameters**
  - Display of current governance parameters
  - Deposit requirements
  - Voting periods
  - Quorum and threshold values

## Architecture

### Components

#### 1. ProposalList Component (`components/ProposalList.js`)
Displays a filterable, searchable list of governance proposals with visual status indicators and progress bars.

**Key Methods:**
- `render(proposals)` - Renders the proposal list
- `renderProposalCard(proposal)` - Creates individual proposal cards
- `calculateProgress(proposal)` - Computes voting percentages
- `getStatusInfo(status)` - Returns status styling information
- `getProposalType(typeString)` - Identifies proposal type

#### 2. ProposalDetail Component (`components/ProposalDetail.js`)
Shows comprehensive information about a specific proposal including timeline, votes, deposits, and tally results.

**Key Methods:**
- `render(proposal)` - Renders full proposal details
- `renderTimeline(proposal)` - Creates visual timeline
- `renderTallyStats(tally)` - Displays voting statistics
- `renderDepositProgress(proposal, deposits)` - Shows deposit progress
- `handleVote(option)` - Initiates voting process

#### 3. CreateProposal Component (`components/CreateProposal.js`)
Provides a form interface for creating new governance proposals with type-specific fields and validation.

**Key Methods:**
- `render()` - Renders the proposal creation form
- `handleTypeChange(type)` - Updates form based on proposal type
- `updatePreview()` - Live preview of proposal
- `handleSubmit()` - Validates and submits proposal

#### 4. VotingPanel Component (`components/VotingPanel.js`)
Modal interface for casting votes with vote option descriptions and confirmation.

**Key Methods:**
- `show(proposal, preselectedOption)` - Displays voting modal
- `renderVotingForm()` - Creates voting interface
- `confirmVote()` - Submits vote transaction

#### 5. TallyChart Component (`components/TallyChart.js`)
Renders voting results using Chart.js with multiple visualization types.

**Key Methods:**
- `render(canvasId, tally)` - Creates doughnut chart
- `renderBarChart(canvasId, tally)` - Creates bar chart
- `renderPolarChart(canvasId, tally)` - Creates polar area chart
- `formatVotingPower(amount)` - Formats large numbers

### Services

#### GovernanceAPI (`services/governanceAPI.js`)
Handles all blockchain interactions for governance operations.

**Key Methods:**
- `checkConnection()` - Verifies blockchain connectivity
- `getAllProposals()` - Fetches all proposals
- `getProposal(proposalId)` - Gets specific proposal
- `getProposalVotes(proposalId)` - Retrieves votes
- `getProposalDeposits(proposalId)` - Gets deposits
- `getProposalTally(proposalId)` - Fetches tally results
- `getGovernanceParameters()` - Gets governance params
- `submitProposal(data, deposit, address)` - Submits new proposal
- `vote(proposalId, option, address)` - Casts vote
- `deposit(proposalId, amount, address)` - Adds deposit
- `getUserVotes(address)` - Gets user's voting history

## Installation

### Prerequisites
- Modern web browser with JavaScript enabled
- Access to PAW blockchain RPC/REST endpoints
- Wallet integration (Keplr, MetaMask Snap, or compatible)

### Setup

1. **Deploy the Portal**
   ```bash
   # Copy all files to your web server
   cp -r dashboards/governance /var/www/html/governance
   ```

2. **Configure Endpoints**
   Edit `services/governanceAPI.js`:
   ```javascript
   this.baseURL = 'https://your-rest-endpoint:1317';
   this.rpcURL = 'https://your-rpc-endpoint:26657';
   this.mockMode = false; // Disable mock mode for production
   ```

3. **Open in Browser**
   ```
   http://localhost/governance/index.html
   ```

## Usage

### Viewing Proposals

1. Navigate to the Proposals section (default view)
2. Use filters to narrow down proposals by:
   - Status (Voting, Deposit Period, Passed, Rejected, Failed)
   - Type (Text, Parameter Change, Software Upgrade, Community Spend)
3. Use the search bar to find specific proposals
4. Click on any proposal to view details

### Voting on Proposals

1. Connect your wallet using the "Connect Wallet" button
2. Navigate to a proposal in Voting Period
3. Click "View Details" to see full proposal information
4. Review the proposal timeline, description, and current votes
5. Click one of the vote buttons:
   - **Yes** - Support the proposal
   - **No** - Oppose the proposal
   - **Abstain** - Contribute to quorum without taking a position
   - **No With Veto** - Strongly oppose and potentially burn deposits
6. Review your vote in the modal
7. Add an optional memo
8. Confirm the vote transaction

### Creating Proposals

1. Connect your wallet
2. Navigate to "Create Proposal" section
3. Select proposal type:
   - **Text Proposal** - General purpose signaling
   - **Parameter Change** - Modify blockchain parameters
   - **Software Upgrade** - Schedule network upgrade
   - **Community Pool Spend** - Allocate community funds
4. Fill in required fields:
   - Title (max 140 characters)
   - Description (detailed explanation)
   - Type-specific fields
   - Initial deposit (minimum 10,000 PAW)
5. Preview your proposal
6. Submit the proposal

### Adding Deposits

1. Navigate to a proposal in Deposit Period
2. Click "Add Deposit"
3. Enter deposit amount
4. Confirm the transaction
5. Monitor deposit progress toward minimum threshold

### Viewing Analytics

1. Navigate to Analytics section
2. View:
   - Proposal success rate pie chart
   - Voting trends line chart
   - Participation rate bar chart
   - Top voters list

### Checking Voting History

1. Connect your wallet
2. Navigate to History section
3. View all your past votes with:
   - Proposal ID and title
   - Vote option
   - Timestamp

## Testing

### Automated Tests

Run the comprehensive test suite:

1. **In Browser:**
   ```
   Open: tests/test-runner.html
   Click: "Run All Tests"
   ```

2. **Via Node.js:**
   ```bash
   node tests/governance.test.js
   ```

### Test Coverage

The test suite includes:
- **Unit Tests** (60+ tests)
  - GovernanceAPI methods
  - Component rendering
  - Data formatting
  - Vote calculations
  - XSS prevention

- **Integration Tests**
  - Proposal listing flow
  - Proposal detail flow
  - Create proposal flow
  - Voting flow
  - Deposit flow
  - Parameter retrieval

- **Edge Cases**
  - Empty data handling
  - Null value handling
  - Invalid input handling

### Manual Testing

1. **Proposal Viewing**
   - ✓ All proposals load correctly
   - ✓ Filters work as expected
   - ✓ Search finds proposals
   - ✓ Proposal cards display accurate data

2. **Proposal Details**
   - ✓ Timeline renders correctly
   - ✓ Vote tally chart displays
   - ✓ Votes and deposits list properly
   - ✓ Status indicators accurate

3. **Voting**
   - ✓ Voting modal opens correctly
   - ✓ Vote options selectable
   - ✓ Confirmation required
   - ✓ Success message displays

4. **Proposal Creation**
   - ✓ Form validation works
   - ✓ Type-specific fields show/hide
   - ✓ Preview updates in real-time
   - ✓ Submission succeeds

5. **Analytics**
   - ✓ Charts render correctly
   - ✓ Data accurate
   - ✓ Interactive features work

## API Reference

### Proposal Object Structure
```javascript
{
  proposal_id: "1",
  content: {
    "@type": "/cosmos.gov.v1beta1.TextProposal",
    title: "Proposal Title",
    description: "Detailed description..."
  },
  status: "VOTING_PERIOD",
  final_tally_result: {
    yes: "45000000",
    abstain: "5000000",
    no: "8000000",
    no_with_veto: "2000000"
  },
  submit_time: "2024-01-15T10:00:00Z",
  deposit_end_time: "2024-01-29T10:00:00Z",
  total_deposit: [{ denom: "paw", amount: "10000000" }],
  voting_start_time: "2024-01-20T10:00:00Z",
  voting_end_time: "2024-02-05T10:00:00Z"
}
```

### Vote Options
- `VOTE_OPTION_YES` - Support the proposal
- `VOTE_OPTION_NO` - Oppose the proposal
- `VOTE_OPTION_ABSTAIN` - Abstain from voting
- `VOTE_OPTION_NO_WITH_VETO` - Veto the proposal

### Proposal Statuses
- `DEPOSIT_PERIOD` - Collecting minimum deposit
- `VOTING_PERIOD` - Active voting
- `PASSED` - Proposal passed
- `REJECTED` - Proposal rejected
- `FAILED` - Proposal failed (did not meet quorum)

## Configuration

### Governance Parameters

Default parameters (configurable via governance):

```javascript
{
  deposit: {
    min_deposit: [{ denom: "paw", amount: "10000000" }], // 10,000 PAW
    max_deposit_period: "1209600s" // 14 days
  },
  voting: {
    voting_period: "1209600s" // 14 days
  },
  tally: {
    quorum: "0.334000000000000000", // 33.4%
    threshold: "0.500000000000000000", // 50%
    veto_threshold: "0.334000000000000000" // 33.4%
  }
}
```

## Security

### Best Practices Implemented

1. **XSS Prevention**
   - All user input is HTML-escaped
   - Content Security Policy headers recommended
   - No `eval()` or `innerHTML` with untrusted data

2. **Transaction Safety**
   - Confirmation required for all transactions
   - Clear warnings about irreversible actions
   - Transaction validation before submission

3. **Wallet Security**
   - No private key storage
   - Wallet interactions via standard APIs
   - Transaction signing in wallet extension

4. **Data Validation**
   - All inputs validated client-side
   - Form validation before submission
   - Type checking on all API responses

## Browser Compatibility

- Chrome/Edge 90+
- Firefox 88+
- Safari 14+
- Opera 76+

## Performance

- Lazy loading of proposal details
- Efficient chart rendering
- Optimized API calls
- Responsive design for mobile devices

## Troubleshooting

### Common Issues

**Issue: "Disconnected" status**
- Check blockchain RPC/REST endpoints
- Verify network connectivity
- Confirm endpoints in `governanceAPI.js`

**Issue: Wallet won't connect**
- Install compatible wallet extension
- Unlock wallet
- Refresh page and retry

**Issue: Charts not displaying**
- Ensure Chart.js loaded correctly
- Check browser console for errors
- Verify canvas elements exist

**Issue: Votes not submitting**
- Confirm wallet has sufficient funds for gas
- Check if proposal is in voting period
- Verify wallet connected

## Development

### Mock Mode

For development and testing, mock mode is enabled by default:

```javascript
// services/governanceAPI.js
this.mockMode = true; // Set to false for production
```

Mock mode provides:
- Sample proposals
- Simulated voting
- Mock transaction responses
- No blockchain connection required

### Extending Functionality

To add new proposal types:

1. Update `governanceAPI.js` mock data
2. Add type detection in `ProposalList.getProposalType()`
3. Add form fields in `CreateProposal.handleTypeChange()`
4. Update styling in `styles.css`

## Contributing

Please follow PAW blockchain contribution guidelines when submitting improvements.

## License

Part of the PAW blockchain project.

## Support

For issues and questions:
-  Issues: [PAW Repository]
- Documentation: [PAW Docs]
- Community: [PAW Discord/Telegram]

---

**Version:** 1.0.0
**Last Updated:** 2024-01-19
**Status:** Production Ready
