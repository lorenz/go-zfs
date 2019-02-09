#include <stdint.h>
#include <stdbool.h>
#include <linux/limits.h>

#define MAXPATHLEN PATH_MAX
#define MAXNAMELEN NAME_MAX
#define	ZFS_MAX_DATASET_NAME_LEN 256

typedef bool boolean_t;

typedef enum dmu_objset_type {
	DMU_OST_NONE,
	DMU_OST_META,
	DMU_OST_ZFS,
	DMU_OST_ZVOL,
	DMU_OST_OTHER,			/* For testing only! */
	DMU_OST_ANY,			/* Be careful! */
	DMU_OST_NUMTYPES
} dmu_objset_type_t;

typedef struct dmu_objset_stats {
	uint64_t dds_num_clones; /* number of clones of this */
	uint64_t dds_creation_txg;
	uint64_t dds_guid;
	dmu_objset_type_t dds_type;
	uint8_t dds_is_snapshot;
	uint8_t dds_inconsistent;
	char dds_origin[ZFS_MAX_DATASET_NAME_LEN];
} dmu_objset_stats_t;

typedef struct zinject_record {
	uint64_t	zi_objset;
	uint64_t	zi_object;
	uint64_t	zi_start;
	uint64_t	zi_end;
	uint64_t	zi_guid;
	uint32_t	zi_level;
	uint32_t	zi_error;
	uint64_t	zi_type;
	uint32_t	zi_freq;
	uint32_t	zi_failfast;
	char		zi_func[MAXNAMELEN];
	uint32_t	zi_iotype;
	int32_t		zi_duration;
	uint64_t	zi_timer;
	uint64_t	zi_nlanes;
	uint32_t	zi_cmd;
	uint32_t	zi_pad;
} zinject_record_t;

typedef struct zfs_share {
	uint64_t	z_exportdata;
	uint64_t	z_sharedata;
	uint64_t	z_sharetype;	/* 0 = share, 1 = unshare */
	uint64_t	z_sharemax;  /* max length of share string */
} zfs_share_t;

typedef struct zfs_stat {
	uint64_t	zs_gen;
	uint64_t	zs_mode;
	uint64_t	zs_links;
	uint64_t	zs_ctime[2];
} zfs_stat_t;

struct drr_begin {
	uint64_t drr_magic;
	uint64_t drr_versioninfo; /* was drr_version */
	uint64_t drr_creation_time;
	dmu_objset_type_t drr_type;
	uint32_t drr_flags;
	uint64_t drr_toguid;
	uint64_t drr_fromguid;
	char drr_toname[MAXNAMELEN];
} drr_begin;

typedef struct zfs_cmd {
	char		zc_name[MAXPATHLEN];	/* name of pool or dataset */
	uint64_t	zc_nvlist_src;		/* really (char *) */
	uint64_t	zc_nvlist_src_size;
	uint64_t	zc_nvlist_dst;		/* really (char *) */
	uint64_t	zc_nvlist_dst_size;
	boolean_t	zc_nvlist_dst_filled;	/* put an nvlist in dst? */
	int		zc_pad2;

	/*
	 * The following members are for legacy ioctls which haven't been
	 * converted to the new method.
	 */
	uint64_t	zc_history;		/* really (char *) */
	char		zc_value[MAXPATHLEN * 2];
	char		zc_string[MAXNAMELEN];
	uint64_t	zc_guid;
	uint64_t	zc_nvlist_conf;		/* really (char *) */
	uint64_t	zc_nvlist_conf_size;
	uint64_t	zc_cookie;
	uint64_t	zc_objset_type;
	uint64_t	zc_perm_action;
	uint64_t	zc_history_len;
	uint64_t	zc_history_offset;
	uint64_t	zc_obj;
	uint64_t	zc_iflags;		/* internal to zfs(7fs) */
	zfs_share_t	zc_share;
	dmu_objset_stats_t zc_objset_stats;
	struct drr_begin zc_begin_record;
	zinject_record_t zc_inject_record;
	uint32_t	zc_defer_destroy;
	uint32_t	zc_flags;
	uint64_t	zc_action_handle;
	int		zc_cleanup_fd;
	uint8_t		zc_simple;
	uint8_t		zc_pad[3];		/* alignment */
	uint64_t	zc_sendobj;
	uint64_t	zc_fromobj;
	uint64_t	zc_createtxg;
	zfs_stat_t	zc_stat;
} zfs_cmd_t;

typedef enum zfs_ioc {
	/*
	 * Illumos - 71/128 numbers reserved.
	 */
	ZFS_IOC_FIRST =	('Z' << 8),
	ZFS_IOC = ZFS_IOC_FIRST,
	ZFS_IOC_POOL_CREATE = ZFS_IOC_FIRST,
	ZFS_IOC_POOL_DESTROY,
	ZFS_IOC_POOL_IMPORT,
	ZFS_IOC_POOL_EXPORT,
	ZFS_IOC_POOL_CONFIGS,
	ZFS_IOC_POOL_STATS,
	ZFS_IOC_POOL_TRYIMPORT,
	ZFS_IOC_POOL_SCAN,
	ZFS_IOC_POOL_FREEZE,
	ZFS_IOC_POOL_UPGRADE,
	ZFS_IOC_POOL_GET_HISTORY,
	ZFS_IOC_VDEV_ADD,
	ZFS_IOC_VDEV_REMOVE,
	ZFS_IOC_VDEV_SET_STATE,
	ZFS_IOC_VDEV_ATTACH,
	ZFS_IOC_VDEV_DETACH,
	ZFS_IOC_VDEV_SETPATH,
	ZFS_IOC_VDEV_SETFRU,
	ZFS_IOC_OBJSET_STATS,
	ZFS_IOC_OBJSET_ZPLPROPS,
	ZFS_IOC_DATASET_LIST_NEXT,
	ZFS_IOC_SNAPSHOT_LIST_NEXT,
	ZFS_IOC_SET_PROP,
	ZFS_IOC_CREATE,
	ZFS_IOC_DESTROY,
	ZFS_IOC_ROLLBACK,
	ZFS_IOC_RENAME,
	ZFS_IOC_RECV,
	ZFS_IOC_SEND,
	ZFS_IOC_INJECT_FAULT,
	ZFS_IOC_CLEAR_FAULT,
	ZFS_IOC_INJECT_LIST_NEXT,
	ZFS_IOC_ERROR_LOG,
	ZFS_IOC_CLEAR,
	ZFS_IOC_PROMOTE,
	ZFS_IOC_SNAPSHOT,
	ZFS_IOC_DSOBJ_TO_DSNAME,
	ZFS_IOC_OBJ_TO_PATH,
	ZFS_IOC_POOL_SET_PROPS,
	ZFS_IOC_POOL_GET_PROPS,
	ZFS_IOC_SET_FSACL,
	ZFS_IOC_GET_FSACL,
	ZFS_IOC_SHARE,
	ZFS_IOC_INHERIT_PROP,
	ZFS_IOC_SMB_ACL,
	ZFS_IOC_USERSPACE_ONE,
	ZFS_IOC_USERSPACE_MANY,
	ZFS_IOC_USERSPACE_UPGRADE,
	ZFS_IOC_HOLD,
	ZFS_IOC_RELEASE,
	ZFS_IOC_GET_HOLDS,
	ZFS_IOC_OBJSET_RECVD_PROPS,
	ZFS_IOC_VDEV_SPLIT,
	ZFS_IOC_NEXT_OBJ,
	ZFS_IOC_DIFF,
	ZFS_IOC_TMP_SNAPSHOT,
	ZFS_IOC_OBJ_TO_STATS,
	ZFS_IOC_SPACE_WRITTEN,
	ZFS_IOC_SPACE_SNAPS,
	ZFS_IOC_DESTROY_SNAPS,
	ZFS_IOC_POOL_REGUID,
	ZFS_IOC_POOL_REOPEN,
	ZFS_IOC_SEND_PROGRESS,
	ZFS_IOC_LOG_HISTORY,
	ZFS_IOC_SEND_NEW,
	ZFS_IOC_SEND_SPACE,
	ZFS_IOC_CLONE,
	ZFS_IOC_BOOKMARK,
	ZFS_IOC_GET_BOOKMARKS,
	ZFS_IOC_DESTROY_BOOKMARKS,
	ZFS_IOC_RECV_NEW,
	ZFS_IOC_POOL_SYNC,

	/*
	 * Linux - 3/64 numbers reserved.
	 */
	ZFS_IOC_LINUX = ('Z' << 8) + 0x80,
	ZFS_IOC_EVENTS_NEXT,
	ZFS_IOC_EVENTS_CLEAR,
	ZFS_IOC_EVENTS_SEEK,

	/*
	 * FreeBSD - 1/64 numbers reserved.
	 */
	ZFS_IOC_FREEBSD = ('Z' << 8) + 0xC0,

	ZFS_IOC_LAST
} zfs_ioc_t;
