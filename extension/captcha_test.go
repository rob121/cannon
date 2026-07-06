package extension

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRegisterCaptchaCapability(t *testing.T) {
	s := New(Info{Name: "cannon-ext-captcha-cfturnstile", Title: "Turnstile", Version: "1"})
	s.RegisterCaptcha(CaptchaRegistration{
		Render: func(req WireRequest) (CaptchaRenderResult, error) {
			if CaptchaContext(req) != CaptchaContextLogin {
				t.Fatalf("context = %q", CaptchaContext(req))
			}
			return CaptchaRenderResult{
				HTML:      `<div class="turnstile"></div>`,
				FieldName: "cf-turnstile-response",
			}, nil
		},
		Verify: func(req WireRequest) (CaptchaVerifyResult, error) {
			if CaptchaToken(req) != "good-token" {
				return CaptchaVerifyResult{Valid: false, Error: "invalid token"}, nil
			}
			return CaptchaVerifyResult{Valid: true}, nil
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/capabilities", nil)
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
	var caps capabilitiesResponse
	if err := json.NewDecoder(rec.Body).Decode(&caps); err != nil {
		t.Fatal(err)
	}
	if caps.Capabilities["captcha"] != "/captcha" {
		t.Fatalf("captcha capability: got %#v", caps.Capabilities)
	}

	req = httptest.NewRequest(http.MethodGet, "/captcha", nil)
	rec = httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
	var info CaptchaProviderInfo
	if err := json.NewDecoder(rec.Body).Decode(&info); err != nil {
		t.Fatal(err)
	}
	if info.ID != "cannon-ext-captcha-cfturnstile" || info.Title != "Turnstile" {
		t.Fatalf("info = %#v", info)
	}

	body := `{"captcha_context":"login","csrf":"abc"}`
	req = httptest.NewRequest(http.MethodPost, "/captcha/render", strings.NewReader(body))
	rec = httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
	var render CaptchaRenderResult
	if err := json.NewDecoder(rec.Body).Decode(&render); err != nil {
		t.Fatal(err)
	}
	if render.FieldName != "cf-turnstile-response" {
		t.Fatalf("render = %#v", render)
	}

	body = `{"captcha_context":"login","captcha_token":"bad"}`
	req = httptest.NewRequest(http.MethodPost, "/captcha/verify", strings.NewReader(body))
	rec = httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("verify bad status = %d", rec.Code)
	}

	body = `{"captcha_context":"login","captcha_token":"good-token","captcha_remote_ip":"127.0.0.1"}`
	req = httptest.NewRequest(http.MethodPost, "/captcha/verify", strings.NewReader(body))
	rec = httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
	var verify CaptchaVerifyResult
	if err := json.NewDecoder(rec.Body).Decode(&verify); err != nil {
		t.Fatal(err)
	}
	if !verify.Valid {
		t.Fatalf("verify = %#v", verify)
	}
}

func TestRegisterCaptchaRequiresHandlers(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic")
		}
	}()
	s := New(Info{Name: "x", Version: "1"})
	s.RegisterCaptcha(CaptchaRegistration{})
}
