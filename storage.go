package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/tencentyun/cos-go-sdk-v5"
)

type avatarUploadResult struct {
	URL    string
	FileID *string
}

type avatarUploader func(ctx context.Context, b []byte, contentType, key string) (avatarUploadResult, error)

type cosConfig struct {
	Bucket       string
	Region       string
	CloudEnv     string
	PublicDomain string
}

// wxCosUploader uploads to tencent COS with object-level public-read ACL and
// returns the COS public URL directly. Temp credentials come from the WeChat
// platform's auth-free endpoint (no AK/SK in the container). Mirrors the TS
// wxCosUploader. Returns a wx cloud fileId when CloudEnv is set.
func wxCosUploader(cfg cosConfig) avatarUploader {
	host := cfg.PublicDomain
	if host == "" {
		host = fmt.Sprintf("%s.cos.%s.myqcloud.com", cfg.Bucket, cfg.Region)
	}
	bucketURL, _ := url.Parse(fmt.Sprintf("https://%s.cos.%s.myqcloud.com", cfg.Bucket, cfg.Region))

	return func(ctx context.Context, body []byte, contentType, key string) (avatarUploadResult, error) {
		cred, err := fetchWxCosAuth(ctx)
		if err != nil {
			return avatarUploadResult{}, err
		}
		client := cos.NewClient(&cos.BaseURL{BucketURL: bucketURL}, &http.Client{
			Transport: &cos.CredentialTransport{
				Credential: cos.NewTokenCredential(cred.TmpSecretID, cred.TmpSecretKey, cred.Token),
			},
		})
		opt := &cos.ObjectPutOptions{
			ObjectPutHeaderOptions: &cos.ObjectPutHeaderOptions{ContentType: contentType},
			ACLHeaderOptions:       &cos.ACLHeaderOptions{XCosACL: "public-read"},
		}
		if _, err := client.Object.Put(ctx, key, bytes.NewReader(body), opt); err != nil {
			return avatarUploadResult{}, err
		}
		result := avatarUploadResult{
			URL: fmt.Sprintf("https://%s/%s?v=%d", host, key, time.Now().UnixMilli()),
		}
		if cfg.CloudEnv != "" {
			fileID := fmt.Sprintf("cloud://%s.%s/%s", cfg.CloudEnv, cfg.Bucket, key)
			result.FileID = &fileID
		}
		return result, nil
	}
}

type wxCosAuth struct {
	TmpSecretID  string `json:"TmpSecretId"`
	TmpSecretKey string `json:"TmpSecretKey"`
	Token        string `json:"Token"`
	ExpiredTime  int64  `json:"ExpiredTime"`
}

func fetchWxCosAuth(ctx context.Context) (*wxCosAuth, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://api.weixin.qq.com/_/cos/getauth", nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var auth wxCosAuth
	if err := json.NewDecoder(resp.Body).Decode(&auth); err != nil {
		return nil, err
	}
	if auth.TmpSecretID == "" {
		return nil, fmt.Errorf("wx cos getauth returned empty credentials")
	}
	return &auth, nil
}

var dataURLRe = regexp.MustCompile(`^data:(image/(?:png|jpeg|jpg|webp));base64,(.+)$`)

type decodedDataURL struct {
	Bytes       []byte
	ContentType string
}

// decodeDataURL parses a `data:image/...;base64,xxx` URL into raw bytes,
// capping at ~2MB. Returns nil on malformed input or oversized payload.
func decodeDataURL(dataURL string) *decodedDataURL {
	m := dataURLRe.FindStringSubmatch(dataURL)
	if m == nil {
		return nil
	}
	contentType := m[1]
	if contentType == "image/jpg" {
		contentType = "image/jpeg"
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(m[2]))
	if err != nil {
		return nil
	}
	if len(raw) > 2*1024*1024 {
		return nil
	}
	return &decodedDataURL{Bytes: raw, ContentType: contentType}
}

func extForContentType(ct string) string {
	if ct == "image/png" {
		return "png"
	}
	return "jpg"
}
