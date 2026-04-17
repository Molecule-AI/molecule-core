package handlers

// Nous Research recommended sampling defaults for Hermes models.
// Applied when the caller does not supply an explicit value for that parameter.
//
// Reference: https://nousresearch.com/hermes3/ — "Recommended Settings"
const (
	hermesDefaultTemperature       = 0.7
	hermesDefaultTopP              = 0.9
	hermesDefaultTopK              = 50
	hermesDefaultRepetitionPenalty = 1.1
)

// HermesSamplingParams holds optional sampling parameters for a vLLM /
// OpenRouter chat-completion request.
//
// Every field is a pointer to distinguish "caller did not supply this value"
// (nil) from "caller explicitly set this value" (non-nil, including *0.0).
// This is critical for temperature: 0.0 is a valid greedy-decoding request
// and must pass through unchanged rather than being replaced by the 0.7
// default.  A bare float64 field cannot make this distinction.
type HermesSamplingParams struct {
	Temperature       *float64
	TopP              *float64
	TopK              *int
	RepetitionPenalty *float64
}

// applyHermesDefaults fills in nil (absent) fields of p with the Nous
// Research recommended sampling defaults for Hermes models.
//
// Non-nil fields — including explicitly-supplied zero values such as
// temperature=0.0 for greedy decoding — are left unchanged so caller
// overrides always take effect.
//
// Each default is copied into a fresh local variable before taking its
// address so that each HermesSamplingParams instance holds an independent
// copy; mutating one instance does not affect another.
func applyHermesDefaults(p *HermesSamplingParams) {
	if p.Temperature == nil {
		v := hermesDefaultTemperature
		p.Temperature = &v
	}
	if p.TopP == nil {
		v := hermesDefaultTopP
		p.TopP = &v
	}
	if p.TopK == nil {
		v := hermesDefaultTopK
		p.TopK = &v
	}
	if p.RepetitionPenalty == nil {
		v := hermesDefaultRepetitionPenalty
		p.RepetitionPenalty = &v
	}
}

// applyHermesDefaultsToBody fills in absent sampling parameters in an
// OpenAI-compatible vLLM request body (map[string]interface{}) with the
// Nous Research recommended defaults.
//
// "Absent" is defined as the key not existing in the map.  A key that is
// present with value 0.0 (e.g. temperature=0.0 for greedy decoding) is left
// untouched — checking map key existence is the idiomatic Go solution to the
// zero-value ambiguity that pointer fields solve in typed structs.
//
// Only the four Nous-recommended sampling keys are ever written; all other
// keys in body (model, messages, stream, extra_body, etc.) are left alone.
//
// This function is the counterpart of mergeSystemMessages: both are
// pre-flight transforms applied to the OpenAI-compat request body before it
// is forwarded to the upstream vLLM / OpenRouter endpoint.
func applyHermesDefaultsToBody(body map[string]interface{}) {
	if _, ok := body["temperature"]; !ok {
		body["temperature"] = hermesDefaultTemperature
	}
	if _, ok := body["top_p"]; !ok {
		body["top_p"] = hermesDefaultTopP
	}
	if _, ok := body["top_k"]; !ok {
		body["top_k"] = hermesDefaultTopK
	}
	if _, ok := body["repetition_penalty"]; !ok {
		body["repetition_penalty"] = hermesDefaultRepetitionPenalty
	}
}
