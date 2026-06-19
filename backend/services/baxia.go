package services

import (
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"time"
)

// Baxia anti-bot cookie generator for chat.qwen.ai
// Implements the ssxmod_itna / ssxmod_itna2 cookie mechanism used by Alibaba Baxia WAF.

const (
	baxiaSDKVersion   = "2.5.36"   // bx-v header value
	baxiaClientSource = "web"      // source header value
	baxiaClientVer    = "0.2.63"   // version header value
	ssxmodRefresh     = 15 * time.Minute
)

// Custom Base64 character table used by Baxia SDK (NOT standard Base64).
const customB64Chars = "DGi0YA7BemWnQjCl4_bR3f8SKIF9tUz/xhr2oEOgPpac=61ZqwTudLkM5vHyNXsVJ"

// ---------- global ssxmod state ----------

var (
	ssxmodMu      sync.RWMutex
	ssxmodItna    string
	ssxmodItna2   string
	ssxmodStarted bool
)

// InitBaxia initialises the ssxmod cookie manager.
// It generates the first cookie immediately and starts a background refresh.
func InitBaxia() {
	ssxmodMu.Lock()
	if ssxmodStarted {
		ssxmodMu.Unlock()
		return
	}
	ssxmodStarted = true
	ssxmodMu.Unlock()

	refreshSsxmod()
	go func() {
		t := time.NewTicker(ssxmodRefresh)
		defer t.Stop()
		for range t.C {
			refreshSsxmod()
		}
	}()
}

func refreshSsxmod() {
	itna, itna2 := generateSsxmodCookies()
	ssxmodMu.Lock()
	ssxmodItna = itna
	ssxmodItna2 = itna2
	ssxmodMu.Unlock()
}

// SsxmodCookies returns the current ssxmod_itna and ssxmod_itna2 values.
func SsxmodCookies() (itna, itna2 string) {
	ssxmodMu.RLock()
	defer ssxmodMu.RUnlock()
	return ssxmodItna, ssxmodItna2
}

// ---------- fingerprint generation ----------

type fingerprintConfig struct {
	DeviceID      string
	Platform      string // "Win32" | "MacIntel" | "Linux x86_64"
	WebGLRenderer string
	Vendor        string
	ScreenInfo    string
	Language      string
	TZOffset      string
}

var defaultFingerprint = fingerprintConfig{
	Platform:      "Win32",
	WebGLRenderer: "ANGLE (NVIDIA, NVIDIA GeForce RTX 3080 Direct3D11 vs_5_0 ps_5_0, D3D11)|Google Inc. (NVIDIA)",
	Vendor:        "Google Inc.",
	ScreenInfo:    "1920|1080|283|1080|158|0|1920|1080|1920|922|0|0",
	Language:      "zh-CN",
	TZOffset:      "-480",
}

func generateDeviceID() string {
	const hex = "0123456789abcdef"
	b := make([]byte, 20)
	for i := range b {
		b[i] = hex[rand.Intn(16)]
	}
	return string(b)
}

func randomHash() int64 {
	return rand.Int63n(4294967296)
}

// buildFingerprintFields returns the 37 raw fields (before encoding).
func buildFingerprintFields(cfg fingerprintConfig) []string {
	if cfg.DeviceID == "" {
		cfg.DeviceID = generateDeviceID()
	}
	ts := time.Now().UnixMilli()
	pluginHash := randomHash()
	canvasHash := randomHash()
	uaHash1 := randomHash()
	uaHash2 := randomHash()
	urlHash := randomHash()
	docHash := rand.Intn(91) + 10

	return []string{
		cfg.DeviceID,                             // 0
		"websdk-2.3.15d",                        // 1  SDK version
		fmt.Sprintf("%d", ts),                    // 2  init timestamp
		"91",                                     // 3
		"1|15",                                   // 4
		cfg.Language,                             // 5
		cfg.TZOffset,                             // 6
		"16705151|12791",                         // 7  color depth
		cfg.ScreenInfo,                           // 8
		"5",                                      // 9
		cfg.Platform,                             // 10
		"10",                                     // 11
		cfg.WebGLRenderer,                        // 12
		"30|30",                                  // 13
		"0",                                      // 14
		"28",                                     // 15
		fmt.Sprintf("5|%d", pluginHash),          // 16
		fmt.Sprintf("%d", canvasHash),            // 17
		fmt.Sprintf("%d", uaHash1),               // 18
		"1",                                      // 19
		"0",                                      // 20
		"1",                                      // 21
		"0",                                      // 22
		"P",                                      // 23  mode
		"0",                                      // 24
		"0",                                      // 25
		"0",                                      // 26
		"416",                                    // 27
		cfg.Vendor,                               // 28
		"8",                                      // 29
		"-1|0|0|0|0",                             // 30
		fmt.Sprintf("%d", uaHash2),               // 31
		"11",                                     // 32
		fmt.Sprintf("%d", time.Now().UnixMilli()), // 33  current timestamp
		fmt.Sprintf("%d", urlHash),               // 34
		"0",                                      // 35
		fmt.Sprintf("%d", docHash),               // 36
	}
}

// ---------- LZW compression (6-bit, custom base64) ----------

func lzwCompress6bit(data string) string {
	if data == "" {
		return ""
	}

	dict := make(map[string]int)
	dictToCreate := make(map[string]bool)
	dictSize := 3
	numBits := 2
	enlargeIn := 2

	var result []byte
	value := 0
	position := 0

	emit := func(v int) {
		result = append(result, customB64Chars[v])
	}

	writeBits := func(n int, bits int) {
		for j := 0; j < bits; j++ {
			value = (value << 1) | (n & 1)
			if position == 5 { // bits-1 where bits=6
				position = 0
				emit(value)
				value = 0
			} else {
				position++
			}
			n >>= 1
		}
	}

	writeChar := func(c string) {
		r := []rune(c)
		if len(r) == 0 {
			return
		}
		code := int(r[0])
		if code < 256 {
			writeBits(0, numBits)
			writeBits(code, 8)
		} else {
			writeBits(1, numBits)
			writeBits(code, 16)
		}
		enlargeIn--
		if enlargeIn == 0 {
			enlargeIn = 1 << numBits
			numBits++
		}
	}

	w := ""
	for i := 0; i < len(data); i++ {
		c := string(data[i])

		if _, ok := dict[c]; !ok {
			dict[c] = dictSize
			dictSize++
			dictToCreate[c] = true
		}

		wc := w + c
		if _, ok := dict[wc]; ok {
			w = wc
		} else {
			if dictToCreate[w] {
				writeChar(w)
				delete(dictToCreate, w)
			} else {
				writeBits(dict[w], numBits)
			}
			enlargeIn--
			if enlargeIn == 0 {
				enlargeIn = 1 << numBits
				numBits++
			}
			dict[wc] = dictSize
			dictSize++
			w = c
		}
	}

	if w != "" {
		if dictToCreate[w] {
			writeChar(w)
			delete(dictToCreate, w)
		} else {
			writeBits(dict[w], numBits)
		}
		enlargeIn--
		if enlargeIn == 0 {
			enlargeIn = 1 << numBits
			numBits++
		}
	}

	// end marker
	writeBits(2, numBits)

	// flush remaining bits
	for {
		value = value << 1
		if position == 5 {
			emit(value)
			break
		}
		position++
	}

	return string(result)
}

// ---------- cookie generation ----------

func generateSsxmodCookies() (itna, itna2 string) {
	fields := buildFingerprintFields(defaultFingerprint)

	// Build ssxmod_itna: all 37 fields joined by ^
	itnaRaw := strings.Join(fields, "^")
	itna = "1-" + lzwCompress6bit(itnaRaw)

	// Build ssxmod_itna2: 18 fields
	itna2Fields := []string{
		fields[0],  // device ID
		fields[1],  // SDK version
		fields[23], // mode (P)
		"0", "", "0", "", "", "0",
		"0", "0",
		fields[32], // constant (11)
		fields[33], // current timestamp
		"0", "0", "0", "0", "0",
	}
	itna2Raw := strings.Join(itna2Fields, "^")
	itna2 = "1-" + lzwCompress6bit(itna2Raw)

	return itna, itna2
}

// BaxiaHeaders returns the extra headers required by Baxia WAF.
// The caller should merge these into the request headers.
func BaxiaHeaders(token string) map[string]string {
	itna, itna2 := SsxmodCookies()
	cookie := "token=" + token
	if itna != "" {
		cookie += ";ssxmod_itna=" + itna
	}
	if itna2 != "" {
		cookie += ";ssxmod_itna2=" + itna2
	}

	tz := time.Now().Format("Mon Jan 02 2006 15:04:05 GMT-0700")

	return map[string]string{
		"bx-v":              baxiaSDKVersion,
		"source":            baxiaClientSource,
		"version":           baxiaClientVer,
		"timezone":          tz,
		"cookie":            cookie,
		"x-accel-buffering": "no",
		"accept-encoding":   "gzip, deflate, br, zstd",
	}
}
