// Graceful-error 渠道测试：jina 不支持 Claude Messages API，应返错误不 panic
// v1.2.0-yuxin 商用化补齐（客户测试前 i 方已测，响应侧需客户 key 自测）
package jina

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
			t.Fatalf("jina ConvertClaudeRequest panic（应优雅报错）: %v", r)
		}
	}()
	a := &Adaptor{}
	result, err := a.ConvertClaudeRequest(&gin.Context{}, &relaycommon.RelayInfo{}, &dto.ClaudeRequest{
		Model: "claude-test",
	})
	if err == nil {
		t.Fatal("jina ConvertClaudeRequest 期望返错误，返 nil")
	}
	if result != nil {
		t.Errorf("jina ConvertClaudeRequest 返 error 时 result 应为 nil，got %v", result)
	}
	if !strings.Contains(err.Error(), "support") {
		t.Errorf("jina 错误信息应含 'support' 关键字：%s", err.Error())
	}

}
