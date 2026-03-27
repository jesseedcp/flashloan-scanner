package aave

import (
	"strings"
	"sync"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

const poolABIJSON = `[
  {
    "type":"function",
    "name":"flashLoan",
    "stateMutability":"nonpayable",
    "inputs":[
      {"name":"receiverAddress","type":"address"},
      {"name":"assets","type":"address[]"},
      {"name":"amounts","type":"uint256[]"},
      {"name":"interestRateModes","type":"uint256[]"},
      {"name":"onBehalfOf","type":"address"},
      {"name":"params","type":"bytes"},
      {"name":"referralCode","type":"uint16"}
    ],
    "outputs":[]
  },
  {
    "type":"function",
    "name":"flashLoanSimple",
    "stateMutability":"nonpayable",
    "inputs":[
      {"name":"receiverAddress","type":"address"},
      {"name":"asset","type":"address"},
      {"name":"amount","type":"uint256"},
      {"name":"params","type":"bytes"},
      {"name":"referralCode","type":"uint16"}
    ],
    "outputs":[]
  },
  {
    "type":"event",
    "name":"FlashLoan",
    "anonymous":false,
    "inputs":[
      {"name":"target","type":"address","indexed":true},
      {"name":"asset","type":"address","indexed":true},
      {"name":"initiator","type":"address","indexed":false},
      {"name":"amount","type":"uint256","indexed":false},
      {"name":"interestRateMode","type":"uint8","indexed":false},
      {"name":"premium","type":"uint256","indexed":false},
      {"name":"referralCode","type":"uint16","indexed":true}
    ]
  }
]`

var (
	poolABI     abi.ABI
	poolABIOnce sync.Once
	poolABIErr  error
)

func PoolABI() (abi.ABI, error) {
	poolABIOnce.Do(func() {
		poolABI, poolABIErr = abi.JSON(strings.NewReader(poolABIJSON))
	})
	return poolABI, poolABIErr
}
