package provider

import (
	"context"
	"net/url"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/payment"
)

// 回归（官方 issue #7881）：利用 return_url 拼接走私 + 复用下单签名
// 伪造支付成功回调的链路必须被拒绝。
func TestVerifyNotificationRejectsForgedSignatureReplay(t *testing.T) {
	t.Parallel()

	e := &EasyPay{config: map[string]string{
		"pid":       "1000",
		"pkey":      "MERCHANT_SECRET_KEY",
		"apiBase":   "https://pay.example.com",
		"notifyUrl": "https://site.example.com/api/v1/payment/webhook/easypay",
	}}

	// 1. 攻击者下单：return_url 尾部藏 &trade_status=TRADE_SUCCESS
	//    （旧代码原样保留进签名；popup 模式下该签名泄露在支付 URL 中）
	returnURL := "https://site.example.com/payment/result?order_id=99&out_trade_no=ORDER123&status=success&trade_status=TRADE_SUCCESS"
	createParams := map[string]string{
		"pid":          "1000",
		"type":         "alipay",
		"out_trade_no": "ORDER123",
		"notify_url":   e.config["notifyUrl"],
		"return_url":   returnURL,
		"name":         "balance recharge",
		"money":        "650.00",
	}
	sign := easyPaySign(createParams, e.config["pkey"])

	// 2. 伪造回调：return_url 只编码前半段，trade_status 以裸 & 提为顶层参数
	prefix := "https://site.example.com/payment/result?order_id=99&out_trade_no=ORDER123&status=success"
	cb := url.Values{}
	cb.Set("pid", "1000")
	cb.Set("type", "alipay")
	cb.Set("out_trade_no", "ORDER123")
	cb.Set("notify_url", e.config["notifyUrl"])
	cb.Set("name", "balance recharge")
	cb.Set("money", "650.00")
	cb.Set("return_url", prefix)
	raw := cb.Encode() + "&trade_status=TRADE_SUCCESS" + "&sign=" + sign + "&sign_type=MD5"

	n, err := e.VerifyNotification(context.Background(), raw, nil)
	if err == nil {
		t.Fatalf("伪造回调被接受（漏洞复现）：status=%v amount=%v", n.Status, n.Amount)
	}
}

// 白名单拒绝任何异常字段（拼接走私的载体字段）。
func TestVerifyNotificationRejectsUnexpectedParams(t *testing.T) {
	t.Parallel()

	e := &EasyPay{config: map[string]string{"pid": "1000", "pkey": "MERCHANT_SECRET_KEY"}}
	params := map[string]string{
		"pid": "1000", "out_trade_no": "ORDER123", "money": "10.00",
		"trade_status": "TRADE_SUCCESS", "device": "mobile",
	}
	sign := easyPaySign(params, e.config["pkey"])
	q := url.Values{}
	for k, v := range params {
		q.Set(k, v)
	}
	raw := q.Encode() + "&sign=" + sign + "&sign_type=MD5"

	if _, err := e.VerifyNotification(context.Background(), raw, nil); err == nil {
		t.Fatal("携带白名单外字段的通知必须被拒绝")
	}
}

// AliMPay（v1 md5）真实通知字段集必须通过，不得被白名单误伤。
func TestVerifyNotificationAcceptsAliMPayNotify(t *testing.T) {
	t.Parallel()

	e := &EasyPay{config: map[string]string{"pid": "1705943590", "pkey": "MERCHANT_SECRET_KEY"}}
	params := map[string]string{
		"pid":          "1705943590",
		"trade_no":     "20261006062655421475",
		"out_trade_no": "PAY-267-123",
		"api_trade_no": "2026100622001412345678901234",
		"type":         "alipay",
		"trade_status": "TRADE_SUCCESS",
		"addtime":      "2026-10-06 16:26:43",
		"endtime":      "2026-10-06 16:27:50",
		"name":         "余额充值",
		"money":        "10.00",
		"buyer":        "test@example.com",
	}
	sign := easyPaySign(params, e.config["pkey"])
	q := url.Values{}
	for k, v := range params {
		q.Set(k, v)
	}
	raw := q.Encode() + "&sign=" + sign + "&sign_type=MD5"

	n, err := e.VerifyNotification(context.Background(), raw, nil)
	if err != nil {
		t.Fatalf("AliMPay 合法通知被拒绝: %v", err)
	}
	if n.Status != payment.ProviderStatusSuccess {
		t.Fatalf("status = %v, want success", n.Status)
	}
	if n.Amount != 10 {
		t.Fatalf("amount = %v, want 10", n.Amount)
	}
	if n.OrderID != "PAY-267-123" {
		t.Fatalf("orderID = %q, want PAY-267-123", n.OrderID)
	}
}

// EPUSDT（epay 兼容，含 trade_id 变体）真实通知字段集必须通过。
func TestVerifyNotificationAcceptsEpusdtNotify(t *testing.T) {
	t.Parallel()

	e := &EasyPay{config: map[string]string{"pid": "1000", "pkey": "MERCHANT_SECRET_KEY"}}
	params := map[string]string{
		"pid":          "1000",
		"trade_id":     "1000001",
		"trade_no":     "1000001",
		"out_trade_no": "PAY-266-456",
		"type":         "alipay",
		"name":         "余额充值",
		"money":        "50.00",
		"trade_status": "TRADE_SUCCESS",
	}
	sign := easyPaySign(params, e.config["pkey"])
	q := url.Values{}
	for k, v := range params {
		q.Set(k, v)
	}
	raw := q.Encode() + "&sign=" + sign + "&sign_type=MD5"

	n, err := e.VerifyNotification(context.Background(), raw, nil)
	if err != nil {
		t.Fatalf("EPUSDT 合法通知被拒绝: %v", err)
	}
	if n.Status != payment.ProviderStatusSuccess {
		t.Fatalf("status = %v, want success", n.Status)
	}
	if n.Amount != 50 {
		t.Fatalf("amount = %v, want 50", n.Amount)
	}
}
