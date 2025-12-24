#!/bin/bash
# Update Compute module keys with namespace prefix 0x01

set -e

FILE="x/compute/keeper/keys.go"

# Add module namespace constant at the top of var block
sed -i '/^var ($/a\	// ModuleNamespace is the namespace byte for the Compute module (0x01)\n\t// All store keys are prefixed with this byte to prevent collisions with other modules\n\tModuleNamespace = []byte{0x01}\n' "$FILE"

# Update all key prefixes
sed -i 's/= \[\]byte{0x01}/= []byte{0x01, 0x01}/' "$FILE"
sed -i 's/= \[\]byte{0x02}/= []byte{0x01, 0x02}/' "$FILE"
sed -i 's/= \[\]byte{0x03}/= []byte{0x01, 0x03}/' "$FILE"
sed -i 's/= \[\]byte{0x04}/= []byte{0x01, 0x04}/' "$FILE"
sed -i 's/= \[\]byte{0x05}/= []byte{0x01, 0x05}/' "$FILE"
sed -i 's/= \[\]byte{0x06}/= []byte{0x01, 0x06}/' "$FILE"
sed -i 's/= \[\]byte{0x07}/= []byte{0x01, 0x07}/' "$FILE"
sed -i 's/= \[\]byte{0x08}/= []byte{0x01, 0x08}/' "$FILE"
sed -i 's/= \[\]byte{0x09}/= []byte{0x01, 0x09}/' "$FILE"
sed -i 's/= \[\]byte{0x0A}/= []byte{0x01, 0x0A}/' "$FILE"
sed -i 's/= \[\]byte{0x0B}/= []byte{0x01, 0x0B}/' "$FILE"
sed -i 's/= \[\]byte{0x0C}/= []byte{0x01, 0x0C}/' "$FILE"
sed -i 's/= \[\]byte{0x0D}/= []byte{0x01, 0x0D}/' "$FILE"
sed -i 's/= \[\]byte{0x0E}/= []byte{0x01, 0x0E}/' "$FILE"
sed -i 's/= \[\]byte{0x0F}/= []byte{0x01, 0x0F}/' "$FILE"
sed -i 's/= \[\]byte{0x10}/= []byte{0x01, 0x10}/' "$FILE"
sed -i 's/= \[\]byte{0x11}/= []byte{0x01, 0x11}/' "$FILE"
sed -i 's/= \[\]byte{0x12}/= []byte{0x01, 0x12}/' "$FILE"
sed -i 's/= \[\]byte{0x13}/= []byte{0x01, 0x13}/' "$FILE"
sed -i 's/= \[\]byte{0x14}/= []byte{0x01, 0x14}/' "$FILE"
sed -i 's/= \[\]byte{0x15}/= []byte{0x01, 0x15}/' "$FILE"
sed -i 's/= \[\]byte{0x16}/= []byte{0x01, 0x16}/' "$FILE"
sed -i 's/= \[\]byte{0x17}/= []byte{0x01, 0x17}/' "$FILE"
sed -i 's/= \[\]byte{0x18}/= []byte{0x01, 0x18}/' "$FILE"
sed -i 's/= \[\]byte{0x19}/= []byte{0x01, 0x19}/' "$FILE"
sed -i 's/= \[\]byte{0x1A}/= []byte{0x01, 0x1A}/' "$FILE"
sed -i 's/= \[\]byte{0x1B}/= []byte{0x01, 0x1B}/' "$FILE"
sed -i 's/= \[\]byte{0x1C}/= []byte{0x01, 0x1C}/' "$FILE"
sed -i 's/= \[\]byte{0x1D}/= []byte{0x01, 0x1D}/' "$FILE"
sed -i 's/= \[\]byte{0x1E}/= []byte{0x01, 0x1E}/' "$FILE"
sed -i 's/= \[\]byte{0x1F}/= []byte{0x01, 0x1F}/' "$FILE"
sed -i 's/= \[\]byte{0x23}/= []byte{0x01, 0x23}/' "$FILE"
sed -i 's/= \[\]byte{0x24}/= []byte{0x01, 0x24}/' "$FILE"
sed -i 's/= \[\]byte{0x25}/= []byte{0x01, 0x25}/' "$FILE"
sed -i 's/= \[\]byte{0x26}/= []byte{0x01, 0x26}/' "$FILE"
sed -i 's/= \[\]byte{0x27}/= []byte{0x01, 0x27}/' "$FILE"

echo "Compute module keys updated successfully"
