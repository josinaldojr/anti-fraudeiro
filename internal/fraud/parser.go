package fraud

import (
	"bytes"
	"fmt"
	"unsafe"
)

func ParseFraudScoreRequest(data []byte, request *FraudScoreRequest) error {
	request.Reset()

	if len(data) == 0 {
		return fmt.Errorf("empty request")
	}

	data = skipWS(data)
	if len(data) == 0 || data[0] != '{' {
		return fmt.Errorf("expected '{'")
	}

	valueEnd := findValueEnd(data)
	if valueEnd <= 0 || valueEnd > len(data) {
		return fmt.Errorf("invalid request object")
	}

	if err := parseObjectInternal(data[:valueEnd], request, ""); err != nil {
		return err
	}
	if len(skipWS(data[valueEnd:])) != 0 {
		return fmt.Errorf("unexpected trailing JSON")
	}

	return nil
}

func parseObjectInternal(data []byte, request *FraudScoreRequest, prefix string) error {
	data = skipWS(data)
	if len(data) == 0 || data[0] != '{' {
		return fmt.Errorf("expected '{'")
	}
	data = data[1:]

	first := true
	for {
		data = skipWS(data)
		if len(data) == 0 {
			return fmt.Errorf("unexpected EOF")
		}
		if data[0] == '}' {
			return nil
		}

		if !first {
			if data[0] != ',' {
				return fmt.Errorf("expected ','")
			}
			data = data[1:]
			data = skipWS(data)
		}
		first = false

		if len(data) == 0 || data[0] != '"' {
			return fmt.Errorf("expected key")
		}

		key, value, rest, err := nextKeyValueInternal(data)
		if err != nil {
			return err
		}
		data = rest

		switch prefix {
		case "":
			switch key {
			case "id":
				request.ID = bytesToString(value)
			case "transaction":
				if err := parseObjectInternal(value, request, "transaction"); err != nil {
					return err
				}
			case "customer":
				if err := parseObjectInternal(value, request, "customer"); err != nil {
					return err
				}
			case "merchant":
				if err := parseObjectInternal(value, request, "merchant"); err != nil {
					return err
				}
			case "terminal":
				if err := parseObjectInternal(value, request, "terminal"); err != nil {
					return err
				}
			case "last_transaction":
				if bytes.Equal(value, []byte("null")) {
					request.LastTransaction = nil
					continue
				}
				if request.LastTransaction == nil {
					request.LastTransaction = &LastTransaction{}
				}
				if err := parseObjectInternal(value, request, "last_transaction"); err != nil {
					return err
				}
			}
		case "transaction":
			switch key {
			case "amount":
				request.Transaction.Amount = fastParseFloat(value)
			case "installments":
				request.Transaction.Installments = fastParseInt(value)
			case "requested_at":
				ts, ok := parseTimestampRFC3339UTCBytes(value)
				if !ok {
					return fmt.Errorf("invalid requested_at")
				}
				request.Transaction.RequestedAt = ts
			}
		case "customer":
			switch key {
			case "avg_amount":
				request.Customer.AvgAmount = fastParseFloat(value)
			case "tx_count_24h":
				request.Customer.TxCount24h = fastParseInt(value)
			case "known_merchants":
				if err := parseStringArray(value, &request.Customer.KnownMerchants); err != nil {
					return err
				}
			}
		case "merchant":
			switch key {
			case "id":
				request.Merchant.ID = bytesToString(value)
			case "mcc":
				request.Merchant.MCC = bytesToString(value)
			case "avg_amount":
				request.Merchant.AvgAmount = fastParseFloat(value)
			}
		case "terminal":
			switch key {
			case "is_online":
				request.Terminal.IsOnline = bytes.Equal(value, []byte("true"))
			case "card_present":
				request.Terminal.CardPresent = bytes.Equal(value, []byte("true"))
			case "km_from_home":
				request.Terminal.KMFromHome = fastParseFloat(value)
			}
		case "last_transaction":
			switch key {
			case "timestamp":
				ts, ok := parseTimestampRFC3339UTCBytes(value)
				if !ok {
					return fmt.Errorf("invalid timestamp")
				}
				request.LastTransaction.Timestamp = ts
			case "km_from_current":
				request.LastTransaction.KMFromCurrent = fastParseFloat(value)
			}
		}
	}
}

func nextKeyValueInternal(data []byte) (key string, value []byte, rest []byte, err error) {
	// data starts with "
	keyEnd := bytes.IndexByte(data[1:], '"')
	if keyEnd == -1 {
		return "", nil, nil, fmt.Errorf("expected key end")
	}
	key = bytesToString(data[1 : keyEnd+1])
	data = data[keyEnd+2:]

	data = skipWS(data)
	if len(data) == 0 || data[0] != ':' {
		return "", nil, nil, fmt.Errorf("expected colon")
	}
	data = data[1:]

	data = skipWS(data)
	valueEnd := findValueEnd(data)
	value = data[:valueEnd]
	rest = data[valueEnd:]

	if len(value) >= 2 && value[0] == '"' && value[len(value)-1] == '"' {
		value = value[1 : len(value)-1]
	}

	return key, value, rest, nil
}

func findValueEnd(data []byte) int {
	if len(data) == 0 {
		return 0
	}

	switch data[0] {
	case '"':
		// String
		for i := 1; i < len(data); i++ {
			if data[i] == '"' && data[i-1] != '\\' {
				return i + 1
			}
		}
		return len(data)
	case '{':
		// Object
		depth := 1
		for i := 1; i < len(data); i++ {
			if data[i] == '{' {
				depth++
			} else if data[i] == '}' {
				depth--
				if depth == 0 {
					return i + 1
				}
			}
		}
		return len(data)
	case '[':
		// Array
		depth := 1
		for i := 1; i < len(data); i++ {
			if data[i] == '[' {
				depth++
			} else if data[i] == ']' {
				depth--
				if depth == 0 {
					return i + 1
				}
			}
		}
		return len(data)
	default:
		// Number, boolean or null
		for i := 0; i < len(data); i++ {
			if data[i] == ',' || data[i] == '}' || data[i] == ']' || data[i] == ' ' || data[i] == '\n' || data[i] == '\r' || data[i] == '\t' {
				return i
			}
		}
		return len(data)
	}
}

func skipWS(data []byte) []byte {
	for i := 0; i < len(data); i++ {
		if data[i] != ' ' && data[i] != '\n' && data[i] != '\r' && data[i] != '\t' {
			return data[i:]
		}
	}
	return nil
}

func parseStringArray(data []byte, slice *[]string) error {
	data = skipWS(data)
	if len(data) < 2 || data[0] != '[' || data[len(data)-1] != ']' {
		return fmt.Errorf("expected array")
	}
	data = data[1 : len(data)-1]

	for len(data) > 0 {
		data = skipWS(data)
		if len(data) == 0 {
			break
		}
		if data[0] == ',' {
			data = data[1:]
			data = skipWS(data)
		}
		if len(data) == 0 {
			break
		}

		valueEnd := findValueEnd(data)
		value := data[:valueEnd]
		data = data[valueEnd:]

		if len(value) >= 2 && value[0] == '"' && value[len(value)-1] == '"' {
			*slice = append(*slice, bytesToString(value[1:len(value)-1]))
		}
	}

	return nil
}

func bytesToString(b []byte) string {
	if len(b) == 0 {
		return ""
	}
	return unsafe.String(unsafe.SliceData(b), len(b))
}

func fastParseFloat(b []byte) float64 {
	if len(b) == 0 {
		return 0
	}

	var neg bool
	if b[0] == '-' {
		neg = true
		b = b[1:]
	}

	var val int64
	var dot int = -1
	for i, c := range b {
		if c >= '0' && c <= '9' {
			val = val*10 + int64(c-'0')
		} else if c == '.' {
			dot = i
		}
	}

	res := float64(val)
	if dot != -1 {
		shift := len(b) - 1 - dot
		switch shift {
		case 1:
			res /= 10
		case 2:
			res /= 100
		case 3:
			res /= 1000
		case 4:
			res /= 10000
		default:
			for i := 0; i < shift; i++ {
				res /= 10
			}
		}
	}

	if neg {
		return -res
	}
	return res
}

func fastParseInt(b []byte) int {
	var val int
	for _, c := range b {
		if c >= '0' && c <= '9' {
			val = val*10 + int(c-'0')
		}
	}
	return val
}
