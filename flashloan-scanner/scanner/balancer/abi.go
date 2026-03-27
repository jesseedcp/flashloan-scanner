package balancer

import (
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

const vaultABIJSON = `[
  {
    "inputs": [
      {"internalType":"contract IFlashLoanRecipient","name":"recipient","type":"address"},
      {"internalType":"contract IERC20[]","name":"tokens","type":"address[]"},
      {"internalType":"uint256[]","name":"amounts","type":"uint256[]"},
      {"internalType":"bytes","name":"userData","type":"bytes"}
    ],
    "name": "flashLoan",
    "outputs": [],
    "stateMutability": "nonpayable",
    "type": "function"
  },
  {
    "anonymous": false,
    "inputs": [
      {"indexed": true, "internalType": "contract IFlashLoanRecipient", "name": "recipient", "type": "address"},
      {"indexed": true, "internalType": "contract IERC20", "name": "token", "type": "address"},
      {"indexed": false, "internalType": "uint256", "name": "amount", "type": "uint256"},
      {"indexed": false, "internalType": "uint256", "name": "feeAmount", "type": "uint256"}
    ],
    "name": "FlashLoan",
    "type": "event"
  }
]`

func VaultABI() (abi.ABI, error) {
	return abi.JSON(strings.NewReader(vaultABIJSON))
}
