package uniswapv2

import (
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

const pairABIJSON = `[
  {
    "inputs": [
      {"internalType":"uint256","name":"amount0Out","type":"uint256"},
      {"internalType":"uint256","name":"amount1Out","type":"uint256"},
      {"internalType":"address","name":"to","type":"address"},
      {"internalType":"bytes","name":"data","type":"bytes"}
    ],
    "name":"swap",
    "outputs":[],
    "stateMutability":"nonpayable",
    "type":"function"
  },
  {
    "anonymous":false,
    "inputs":[
      {"indexed":true,"internalType":"address","name":"sender","type":"address"},
      {"indexed":false,"internalType":"uint256","name":"amount0In","type":"uint256"},
      {"indexed":false,"internalType":"uint256","name":"amount1In","type":"uint256"},
      {"indexed":false,"internalType":"uint256","name":"amount0Out","type":"uint256"},
      {"indexed":false,"internalType":"uint256","name":"amount1Out","type":"uint256"},
      {"indexed":true,"internalType":"address","name":"to","type":"address"}
    ],
    "name":"Swap",
    "type":"event"
  }
]`

func PairABI() (abi.ABI, error) {
	return abi.JSON(strings.NewReader(pairABIJSON))
}
