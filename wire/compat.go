package wire

// Config controls optional behaviors for compatibility/conformance.
// Defaults preserve the current library behavior (baseline conformance status).
type Config struct {
    // FillMissingScalarDefaultsOnDecode: when true, populate absent non-repeated
    // scalar and enum fields with their proto3 defaults during decode.
    // Defaults to false to preserve field presence semantics.
    FillMissingScalarDefaultsOnDecode bool
}

var config = Config{
    FillMissingScalarDefaultsOnDecode: true,
}

// SetConfig sets the global wire configuration. Defaults remain zero-valued
// unless explicitly changed by the caller.
func SetConfig(c Config) { config = c }
