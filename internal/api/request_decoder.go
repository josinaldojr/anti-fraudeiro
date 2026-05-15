package api

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"

	"github.com/josinaldojr/anti-fraudeiro/internal/fraud"
)

var (
	errRequestTooLarge = errors.New("request payload too large")
)

var requestPool = sync.Pool{
	New: func() any {
		return &fraud.FraudScoreRequest{
			Customer: fraud.Customer{
				KnownMerchants: make([]string, 0, 64),
			},
			LastTransaction: &fraud.LastTransaction{},
		}
	},
}

var bufferPool = sync.Pool{
	New: func() any {
		b := make([]byte, maxFraudScoreRequestBodyBytes)
		return &b
	},
}

func decodeFraudScoreRequest(w http.ResponseWriter, r *http.Request) (*fraud.FraudScoreRequest, error) {
	bufPtr := bufferPool.Get().(*[]byte)
	defer bufferPool.Put(bufPtr)
	buf := *bufPtr

	n, err := io.ReadFull(r.Body, buf)
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		return nil, err
	}

	if n == int(maxFraudScoreRequestBodyBytes) {
		// Check if there is more data
		var oneByte [1]byte
		if _, err := r.Body.Read(oneByte[:]); err != io.EOF {
			return nil, errRequestTooLarge
		}
	}

	data := buf[:n]
	request := requestPool.Get().(*fraud.FraudScoreRequest)

	if err := fraud.ParseFraudScoreRequest(data, request); err != nil {
		requestPool.Put(request)
		return nil, fmt.Errorf("parse request: %w", err)
	}

	request.Customer.Finalize()
	return request, nil
}

func releaseFraudScoreRequest(request *fraud.FraudScoreRequest) {
	if request != nil {
		requestPool.Put(request)
	}
}

func WarmupPools() {
	const warmupCount = 256
	
	// Warmup request objects
	reqs := make([]*fraud.FraudScoreRequest, warmupCount)
	for i := 0; i < warmupCount; i++ {
		reqs[i] = requestPool.Get().(*fraud.FraudScoreRequest)
	}
	for i := 0; i < warmupCount; i++ {
		releaseFraudScoreRequest(reqs[i])
	}

	// Warmup buffers
	bufs := make([]*[]byte, warmupCount)
	for i := 0; i < warmupCount; i++ {
		bufs[i] = bufferPool.Get().(*[]byte)
	}
	for i := 0; i < warmupCount; i++ {
		bufferPool.Put(bufs[i])
	}
}
