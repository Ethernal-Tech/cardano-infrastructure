package wallet

type CardanoNetworkType byte

const (
	VectorMainNetNetwork CardanoNetworkType = 5
	VectorTestNetNetwork CardanoNetworkType = 4
	PrimeMainNetNetwork  CardanoNetworkType = 3
	PrimeTestNetNetwork  CardanoNetworkType = 2
	MainNetNetwork       CardanoNetworkType = 1
	TestNetNetwork       CardanoNetworkType = 0
)

func (n CardanoNetworkType) GetPrefix() string {
	switch n {
	case PrimeTestNetNetwork:
		return "prime_test"
	case PrimeMainNetNetwork:
		return "prime"
	case VectorTestNetNetwork:
		return "vector_test"
	case VectorMainNetNetwork:
		return "vector"
	case MainNetNetwork:
		return "addr"
	case TestNetNetwork:
		return "addr_test"
	default:
		return "" // not handled but dont raise an error
	}
}

func (n CardanoNetworkType) GetStakePrefix() string {
	switch n {
	case PrimeTestNetNetwork:
		return "stake_prime_test"
	case PrimeMainNetNetwork:
		return "stake_prime"
	case VectorTestNetNetwork:
		return "stake_vector_test"
	case VectorMainNetNetwork:
		return "stake_vector"
	case MainNetNetwork:
		return "stake"
	case TestNetNetwork:
		return "stake_test"
	default:
		return "" // not handled but dont raise an error
	}
}

func (n CardanoNetworkType) IsMainNet() bool {
	return n == MainNetNetwork
}
