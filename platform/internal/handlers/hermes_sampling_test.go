package handlers

import (
	"testing"
)

// fp and ip are convenience constructors for pointer-to-float64 / pointer-to-int
// used throughout these tests.  Defined locally to avoid polluting other test
// files in the package.
func fp(v float64) *float64 { return &v }
func ip(v int) *int         { return &v }

// ============================================================
// applyHermesDefaults — typed struct (pointer fields)
// ============================================================

// TestApplyHermesDefaults_AllAbsent verifies that all four nil fields receive
// the Nous-recommended defaults when no caller values are set.
func TestApplyHermesDefaults_AllAbsent(t *testing.T) {
	p := &HermesSamplingParams{}
	applyHermesDefaults(p)

	if p.Temperature == nil || *p.Temperature != hermesDefaultTemperature {
		t.Errorf("Temperature: got %v, want %v", p.Temperature, hermesDefaultTemperature)
	}
	if p.TopP == nil || *p.TopP != hermesDefaultTopP {
		t.Errorf("TopP: got %v, want %v", p.TopP, hermesDefaultTopP)
	}
	if p.TopK == nil || *p.TopK != hermesDefaultTopK {
		t.Errorf("TopK: got %v, want %v", p.TopK, hermesDefaultTopK)
	}
	if p.RepetitionPenalty == nil || *p.RepetitionPenalty != hermesDefaultRepetitionPenalty {
		t.Errorf("RepetitionPenalty: got %v, want %v", p.RepetitionPenalty, hermesDefaultRepetitionPenalty)
	}
}

// TestApplyHermesDefaults_ExplicitValuesPassThrough verifies that non-nil
// fields are left completely unchanged — caller overrides always win.
func TestApplyHermesDefaults_ExplicitValuesPassThrough(t *testing.T) {
	p := &HermesSamplingParams{
		Temperature:       fp(0.5),
		TopP:              fp(0.8),
		TopK:              ip(40),
		RepetitionPenalty: fp(1.05),
	}
	applyHermesDefaults(p)

	if *p.Temperature != 0.5 {
		t.Errorf("Temperature: got %v, want 0.5", *p.Temperature)
	}
	if *p.TopP != 0.8 {
		t.Errorf("TopP: got %v, want 0.8", *p.TopP)
	}
	if *p.TopK != 40 {
		t.Errorf("TopK: got %v, want 40", *p.TopK)
	}
	if *p.RepetitionPenalty != 1.05 {
		t.Errorf("RepetitionPenalty: got %v, want 1.05", *p.RepetitionPenalty)
	}
}

// TestApplyHermesDefaults_ZeroTemperaturePassThrough is the critical
// regression test: temperature=0.0 (greedy decoding) must NOT be overwritten
// by the 0.7 default.  A bare float64 field would be indistinguishable from
// "not set" — this test validates that the pointer approach handles it correctly.
func TestApplyHermesDefaults_ZeroTemperaturePassThrough(t *testing.T) {
	p := &HermesSamplingParams{
		Temperature: fp(0.0), // explicit greedy decoding
	}
	applyHermesDefaults(p)

	if *p.Temperature != 0.0 {
		t.Errorf("zero temperature was overwritten: got %v, want 0.0", *p.Temperature)
	}
	// Absent fields must still receive their defaults.
	if p.TopP == nil || *p.TopP != hermesDefaultTopP {
		t.Errorf("TopP: got %v, want default %v", p.TopP, hermesDefaultTopP)
	}
	if p.TopK == nil || *p.TopK != hermesDefaultTopK {
		t.Errorf("TopK: got %v, want default %v", p.TopK, hermesDefaultTopK)
	}
	if p.RepetitionPenalty == nil || *p.RepetitionPenalty != hermesDefaultRepetitionPenalty {
		t.Errorf("RepetitionPenalty: got %v, want default %v", p.RepetitionPenalty, hermesDefaultRepetitionPenalty)
	}
}

// TestApplyHermesDefaults_PartialOverride verifies that a mixture of nil /
// non-nil fields is handled correctly: only nil fields receive defaults.
func TestApplyHermesDefaults_PartialOverride(t *testing.T) {
	p := &HermesSamplingParams{
		TopP: fp(0.95), // only TopP explicitly set
	}
	applyHermesDefaults(p)

	if p.Temperature == nil || *p.Temperature != hermesDefaultTemperature {
		t.Errorf("Temperature: got %v, want default %v", p.Temperature, hermesDefaultTemperature)
	}
	if *p.TopP != 0.95 {
		t.Errorf("TopP: got %v, want 0.95 (caller override)", *p.TopP)
	}
	if p.TopK == nil || *p.TopK != hermesDefaultTopK {
		t.Errorf("TopK: got %v, want default %v", p.TopK, hermesDefaultTopK)
	}
	if p.RepetitionPenalty == nil || *p.RepetitionPenalty != hermesDefaultRepetitionPenalty {
		t.Errorf("RepetitionPenalty: got %v, want default %v", p.RepetitionPenalty, hermesDefaultRepetitionPenalty)
	}
}

// TestApplyHermesDefaults_IndependentValueCopies verifies that two separate
// calls to applyHermesDefaults produce independent pointer values — mutating
// one struct's Temperature must not affect another's.
func TestApplyHermesDefaults_IndependentValueCopies(t *testing.T) {
	p1 := &HermesSamplingParams{}
	p2 := &HermesSamplingParams{}
	applyHermesDefaults(p1)
	applyHermesDefaults(p2)

	*p1.Temperature = 99.0
	if *p2.Temperature == 99.0 {
		t.Error("Temperature pointers share memory — expected independent copies")
	}
}

// TestApplyHermesDefaults_DefaultValues verifies the exact Nous-recommended
// constant values, catching accidental typos.
func TestApplyHermesDefaults_DefaultValues(t *testing.T) {
	p := &HermesSamplingParams{}
	applyHermesDefaults(p)

	tests := []struct {
		name string
		got  float64
		want float64
	}{
		{"Temperature", *p.Temperature, 0.7},
		{"TopP", *p.TopP, 0.9},
		{"RepetitionPenalty", *p.RepetitionPenalty, 1.1},
	}
	for _, tc := range tests {
		if tc.got != tc.want {
			t.Errorf("%s default: got %v, want %v", tc.name, tc.got, tc.want)
		}
	}
	if *p.TopK != 50 {
		t.Errorf("TopK default: got %v, want 50", *p.TopK)
	}
}

// ============================================================
// applyHermesDefaultsToBody — map / JSON body approach
// ============================================================

// TestApplyHermesDefaultsToBody_AllAbsent verifies that all four sampling
// keys are added to an empty body map.
func TestApplyHermesDefaultsToBody_AllAbsent(t *testing.T) {
	body := map[string]interface{}{}
	applyHermesDefaultsToBody(body)

	if body["temperature"] != hermesDefaultTemperature {
		t.Errorf("temperature: got %v, want %v", body["temperature"], hermesDefaultTemperature)
	}
	if body["top_p"] != hermesDefaultTopP {
		t.Errorf("top_p: got %v, want %v", body["top_p"], hermesDefaultTopP)
	}
	if body["top_k"] != hermesDefaultTopK {
		t.Errorf("top_k: got %v, want %v", body["top_k"], hermesDefaultTopK)
	}
	if body["repetition_penalty"] != hermesDefaultRepetitionPenalty {
		t.Errorf("repetition_penalty: got %v, want %v", body["repetition_penalty"], hermesDefaultRepetitionPenalty)
	}
}

// TestApplyHermesDefaultsToBody_ExplicitValuesPreserved verifies that all
// four keys present in the body are left unchanged — explicit caller values win.
func TestApplyHermesDefaultsToBody_ExplicitValuesPreserved(t *testing.T) {
	body := map[string]interface{}{
		"temperature":        0.3,
		"top_p":              0.85,
		"top_k":              25,
		"repetition_penalty": 1.05,
	}
	applyHermesDefaultsToBody(body)

	if body["temperature"] != 0.3 {
		t.Errorf("temperature: got %v, want 0.3", body["temperature"])
	}
	if body["top_p"] != 0.85 {
		t.Errorf("top_p: got %v, want 0.85", body["top_p"])
	}
	if body["top_k"] != 25 {
		t.Errorf("top_k: got %v, want 25", body["top_k"])
	}
	if body["repetition_penalty"] != 1.05 {
		t.Errorf("repetition_penalty: got %v, want 1.05", body["repetition_penalty"])
	}
}

// TestApplyHermesDefaultsToBody_ZeroTemperaturePreserved is the critical
// regression test for the body-map approach: temperature=0.0 is present as a
// key in the map, so it must NOT be overwritten by the default.
//
// This is the JSON-level analogue of TestApplyHermesDefaults_ZeroTemperaturePassThrough.
func TestApplyHermesDefaultsToBody_ZeroTemperaturePreserved(t *testing.T) {
	body := map[string]interface{}{
		"temperature": 0.0, // explicit greedy decoding — key IS present in map
	}
	applyHermesDefaultsToBody(body)

	if body["temperature"] != 0.0 {
		t.Errorf("zero temperature was overwritten: got %v, want 0.0", body["temperature"])
	}
	// Other absent keys must still receive defaults.
	if body["top_p"] != hermesDefaultTopP {
		t.Errorf("top_p: got %v, want default %v", body["top_p"], hermesDefaultTopP)
	}
	if body["top_k"] != hermesDefaultTopK {
		t.Errorf("top_k: got %v, want default %v", body["top_k"], hermesDefaultTopK)
	}
	if body["repetition_penalty"] != hermesDefaultRepetitionPenalty {
		t.Errorf("repetition_penalty: got %v, want default %v", body["repetition_penalty"], hermesDefaultRepetitionPenalty)
	}
}

// TestApplyHermesDefaultsToBody_OtherKeysUntouched verifies that the function
// is surgical: only the four sampling keys are ever written; model, messages,
// stream, and any other keys in the body are left completely alone.
func TestApplyHermesDefaultsToBody_OtherKeysUntouched(t *testing.T) {
	body := map[string]interface{}{
		"model":    "hermes-3-llama-3.1-70b",
		"messages": []interface{}{},
		"stream":   true,
	}
	applyHermesDefaultsToBody(body)

	if body["model"] != "hermes-3-llama-3.1-70b" {
		t.Errorf("model key was modified: got %v", body["model"])
	}
	if body["stream"] != true {
		t.Errorf("stream key was modified: got %v", body["stream"])
	}
	// Sampling defaults must have been inserted.
	if _, ok := body["temperature"]; !ok {
		t.Error("temperature was not added to body")
	}
	if _, ok := body["top_p"]; !ok {
		t.Error("top_p was not added to body")
	}
}

// TestApplyHermesDefaultsToBody_PartialOverride verifies the mixed case:
// caller sets some keys, leaves others absent; only absent keys get defaults.
func TestApplyHermesDefaultsToBody_PartialOverride(t *testing.T) {
	body := map[string]interface{}{
		"temperature": 1.0, // custom — absent keys should get defaults
	}
	applyHermesDefaultsToBody(body)

	if body["temperature"] != 1.0 {
		t.Errorf("temperature: got %v, want 1.0 (caller override)", body["temperature"])
	}
	if body["top_p"] != hermesDefaultTopP {
		t.Errorf("top_p: got %v, want default %v", body["top_p"], hermesDefaultTopP)
	}
	if body["top_k"] != hermesDefaultTopK {
		t.Errorf("top_k: got %v, want default %v", body["top_k"], hermesDefaultTopK)
	}
	if body["repetition_penalty"] != hermesDefaultRepetitionPenalty {
		t.Errorf("repetition_penalty: got %v, want default %v", body["repetition_penalty"], hermesDefaultRepetitionPenalty)
	}
}

// TestApplyHermesDefaultsToBody_NilValueKeyPresent verifies an edge case:
// if a key is explicitly set to nil (e.g. via JSON "temperature": null),
// we treat the key as present and do NOT overwrite it.  Passing nil to
// a provider may have a meaning different from "use default."
func TestApplyHermesDefaultsToBody_NilValueKeyPresent(t *testing.T) {
	body := map[string]interface{}{
		"temperature": nil, // explicitly nulled — key is present
	}
	applyHermesDefaultsToBody(body)

	if body["temperature"] != nil {
		t.Errorf("nil temperature was overwritten: got %v", body["temperature"])
	}
}
