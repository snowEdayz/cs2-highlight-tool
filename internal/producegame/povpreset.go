package producegame

import _ "embed"

// PovVPK is the embedded POV HUD vpk asset shipped with the application binary.
// It is dropped into CS2's csgo directory as `pov.vpk` only when the user enables
// POV HUD recording, and only when no existing `pov.vpk` is present (to avoid
// overwriting a user-supplied file). See produce_gameconfig.go for the runtime
// lifecycle (preparePovForProduce / forceRestorePovForProduce).
//
//go:embed assets/pov.vpk
var PovVPK []byte
