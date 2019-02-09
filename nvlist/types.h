#include <stdint.h>

typedef enum {
	DATA_TYPE_UNKNOWN = 0,
	DATA_TYPE_BOOLEAN,
	DATA_TYPE_BYTE,
	DATA_TYPE_INT16,
	DATA_TYPE_UINT16,
	DATA_TYPE_INT32,
	DATA_TYPE_UINT32,
	DATA_TYPE_INT64,
	DATA_TYPE_UINT64,
	DATA_TYPE_STRING,
	DATA_TYPE_BYTE_ARRAY,
	DATA_TYPE_INT16_ARRAY,
	DATA_TYPE_UINT16_ARRAY,
	DATA_TYPE_INT32_ARRAY,
	DATA_TYPE_UINT32_ARRAY,
	DATA_TYPE_INT64_ARRAY,
	DATA_TYPE_UINT64_ARRAY,
	DATA_TYPE_STRING_ARRAY,
	DATA_TYPE_HRTIME,
	DATA_TYPE_NVLIST,
	DATA_TYPE_NVLIST_ARRAY,
	DATA_TYPE_BOOLEAN_VALUE,
	DATA_TYPE_INT8,
	DATA_TYPE_UINT8,
	DATA_TYPE_BOOLEAN_ARRAY,
	DATA_TYPE_INT8_ARRAY,
#if !defined(_KERNEL)
	DATA_TYPE_UINT8_ARRAY,
	DATA_TYPE_DOUBLE
#else
	DATA_TYPE_UINT8_ARRAY
#endif
} data_type_t;

typedef struct nvpair {
	int32_t nvp_size;	/* size of this nvpair */
	int16_t	nvp_name_sz;	/* length of name string */
	int16_t	nvp_reserve;	/* not used */
	int32_t	nvp_value_elem;	/* number of elements for array types */
	data_type_t nvp_type;	/* type of value */
	/* name string */
	/* aligned ptr array for string arrays */
	/* aligned array of data for value */
} nvpair_t;

/* nvlist header */
typedef struct nvlist {
	int32_t		nvl_version;
	uint32_t	nvl_nvflag;	/* persistent flags */
	uint64_t	nvl_priv;	/* ptr to private data if not packed */
	uint32_t	nvl_flag;
	int32_t		nvl_pad;	/* currently not used, for alignment */
} nvlist_t;

typedef struct nvs_header {
	char	nvh_encoding;	/* nvs encoding method */
	char	nvh_endian;	/* nvs endian */
	char	nvh_reserved1;	/* reserved for future use */
	char	nvh_reserved2;	/* reserved for future use */
} nvs_header_t;
