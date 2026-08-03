package replicate

import (
	"strings"
	"testing"

	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/gin-gonic/gin"
)

func TestAdaptor_ConvertClaudeRequest_GracefulError(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("replicate ConvertClaudeRequest panic: %v", r)
		}
	}()
	a := &Adaptor{}
	_, err := a.ConvertClaudeRequest(&gin.Context{}, &relaycommon.RelayInfo{}, &dto.ClaudeRequest{Model: "claude-test"})
	if err == nil {
		t.Fatal("replicate 期望返错误，返 nil")
	}
	if !strings.Contains(err.Error(), "support") {
		t.Errorf("replicate 错误应含 'support'：%s", err.Error())
	}
}
