## Encoding
1. First 4 bytes are nvs_header_t (only ENCODING_NATIVE = 0 supported, BE = 0, LE = 1)
2. Write version (=0) & nvflag if root element
3. Write nvpair_t with zeroed out pointers


Notes
* Finish marker: 4 zero bytes
* NVPair Size is with header and data
* NVPair Name is just the name length after the header including a null termination
* NVPair Content

* First nvs_header_t
* 64 bit -> 8 bytes
* 0x20 -> 32 -> 4 pointers
* nvlistarray 0x60 -> 2 items
* nvlistarray 0x80 -> 3 items (4 pointers per item)

## XDR

nvpair
* size (i32)
* decoded size (i32, irrelevant)
* name (length-coded i32 without null termination)
* type (i32)
* number of items (i32)

* numerics -> native
* byte -> int32
* string -> length-coded (i32, without null byte) and padded to 4 bytes
* boolean -> empty (number of items zero)
* booleanValue -> i32
* nvlist -> version (i32) + flags (i32) + nvpairs + 8 bytes zero padding
