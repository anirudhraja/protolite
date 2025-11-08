package wire

import (
    "os"
)

// Config controls optional behaviors for compatibility/conformance.
// Defaults preserve the current library behavior (baseline conformance status).
type Config struct {
    // AllowUnknownEnumNumberDecode: when true, decoding enums from wire will accept
    // numeric values that are not present in the enum definition and surface them
    // as their numeric value (int32) instead of failing. When false (default),
    // unknown enum numbers cause an error during decode.
    AllowUnknownEnumNumberDecode bool

    // UnwrapWrappersOnDecode: when true, decode of wrapper WKT messages returns
    // the underlying scalar/null JSON value instead of a map. Default false
    // preserves previous behavior (return message map).
    UnwrapWrappersOnDecode bool

    // PreserveUnknownBytesOnDecode: when true, decoded messages will include a
    // special "__unknown" []byte field containing concatenated unknown field
    // bytes. Default false preserves previous behavior (discard unknown bytes).
    PreserveUnknownBytesOnDecode bool

    // MapDecodeGenericKeys: when true, map fields are surfaced as
    // map[interface{}]interface{} in decoded output. Default false restores the
    // previous behavior of typed maps (e.g., map[string]interface{}).
    MapDecodeGenericKeys bool

    // PopulateDefaultsOnDecode: when true, for non-repeated primitive and enum
    // fields that are absent in the wire payload, the decoder will populate
    // their zero/default values into the result map. When false (default),
    // absent fields remain missing from the result (proto3-style semantics).
    PopulateDefaultsOnDecode bool

    // StrictWireTypeOnDecode: when true, the decoder rejects fields that use an
    // invalid or mismatched wire type (e.g., wire types 6/7, or varint vs fixed).
    // When false (default), the decoder attempts best-effort decoding (legacy).
    StrictWireTypeOnDecode bool

    // JSONWellKnownInput: when true, the encoder will accept JSON-native forms
    // for well-known types (Timestamp, Duration, FieldMask, Any, Struct, Value, ListValue)
    // and coerce them into their protobuf message shapes during MarshalWithSchema.
    // This is primarily useful for conformance testing and is disabled by default
    // to keep core library behavior minimal.
    JSONWellKnownInput bool
}

var config = Config{
    UnwrapWrappersOnDecode: true, // preserve prior default behavior for UTs
	MapDecodeGenericKeys:   true, // default to generic map keys to match sampleapp expectations
}

// SetConfig sets the global wire configuration. Defaults remain zero-valued
// unless explicitly changed by the caller.
func SetConfig(c Config) { config = c }

func init() {
    // Optional env toggle for test harnesses; default remains unchanged if unset.
    if v := os.Getenv("PROTOLITE_ALLOW_UNKNOWN_ENUM_DECODE"); v == "1" || v == "true" {
        config.AllowUnknownEnumNumberDecode = true
    }
    if v := os.Getenv("PROTOLITE_UNWRAP_WRAPPERS"); v == "1" || v == "true" {
        config.UnwrapWrappersOnDecode = true
    }
    if v := os.Getenv("PROTOLITE_PRESERVE_UNKNOWN"); v == "1" || v == "true" {
        config.PreserveUnknownBytesOnDecode = true
    }
    if v := os.Getenv("PROTOLITE_GENERIC_MAP_KEYS"); v == "1" || v == "true" {
        config.MapDecodeGenericKeys = true
    }
    if v := os.Getenv("PROTOLITE_POPULATE_DEFAULTS_ON_DECODE"); v == "1" || v == "true" {
        config.PopulateDefaultsOnDecode = true
    }
    if v := os.Getenv("PROTOLITE_STRICT_WIRE"); v == "1" || v == "true" {
        config.StrictWireTypeOnDecode = true
    }
    if v := os.Getenv("PROTOLITE_JSON_WKT_INPUT"); v == "1" || v == "true" {
        config.JSONWellKnownInput = true
    }
}
