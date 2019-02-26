package ioctl

import (
	"errors"
	"fmt"
	"io"
	"os"

	"golang.org/x/sys/unix"
)

// DatasetProps contains all normal props for a dataset
type DatasetProps struct {
	Name string `nvlist:"name,omitempty"`
}

type ACLInheritancePolicy uint64
type DNodeSize uint64
type CanMount uint64

type FilesystemProps struct {
	SnapshotDirectoryEnabled bool                 `nvlist:"snapdir,asuint64"`
	ACLInheritancePolicy     ACLInheritancePolicy `nvlist:"aclinherit,omitempty,default=4"`
	DNodeSize                DNodeSize            `nvlist:"dnodesize,omitempty"`
	Atime                    bool                 `nvlist:"atime,default=true"`
	RelativeAtime            bool                 `nvlist:"relatime"`

	// All props below do nothing here
	Zoned     bool     `nvlist:"zoned"`
	VirusScan bool     `nvlist:"vscan"`
	Overlay   bool     `nvlist:"overlay"`
	CanMount  CanMount `nvlist:"canmount,default=true"`
	Mounted   bool     `nvlist:"mounted"`

	Mountpoint string `nvlist:"mountpoint"`
}

type VolumeProps struct {
}

// PoolProps represents all properties of a zpool
type PoolProps struct {
	Name    string `nvlist:"name,omitempty"`
	Version uint64 `nvlist:"version,omitempty"`
	Comment string `nvlist:"comment,omitempty"`

	// Pool configuration
	AlternativeRoot string   `nvlist:"altroot,omitempty"`
	TemporaryName   string   `nvlist:"tname,omitempty"`
	BootFS          string   `nvlist:"bootfs,omitempty"`
	CacheFile       string   `nvlist:"cachefile,omitempty"`
	ReadOnly        bool     `nvlist:"readonly,omitempty"`
	Multihost       bool     `nvlist:"multihost,omitempty"`
	Failmode        FailMode `nvlist:"failmode,omitempty"`
	DedupDitto      uint64   `nvlist:"dedupditto,omitempty"`
	AlignmentShift  uint64   `nvlist:"ashift,omitempty"`
	Delegation      bool     `nvlist:"delegation,omitempty"`
	Autoreplace     bool     `nvlist:"autoreplace,omitempty"`
	ListSnapshots   bool     `nvlist:"listsnapshots,omitempty"`
	Autoexpand      bool     `nvlist:"autoexpand,omitempty"`
	MaxBlockSize    uint64   `nvlist:"maxblocksize,omitempty"`
	MaxDnodeSize    uint64   `nvlist:"maxdnodesize,omitempty"`

	// Defines props for the root volume for PoolCreate()
	RootProps *DatasetProps `nvlist:"root-props-nvl,omitempty"`

	// All user properties are represented here
	User map[string]string `nvlist:"-,extra,omitempty"`

	// Read-only information
	Size          uint64 `nvlist:"size,ro"`
	Free          uint64 `nvlist:"free,ro"`
	Freeing       uint64 `nvlist:"freeing,ro"`
	Leaked        uint64 `nvlist:"leaked,ro"`
	Allocated     uint64 `nvlist:"allocated,ro"`
	ExpandSize    uint64 `nvlist:"expandsize,ro"`
	Fragmentation uint64 `nvlist:"fragmentation,ro"`
	Capacity      uint64 `nvlist:"capacity,ro"`
	GUID          uint64 `nvlist:"guid,ro"`
	Health        State  `nvlist:"health,ro"`
	DedupRatio    uint64 `nvlist:"dedupratio,ro"`
}

var zfsHandle *os.File

// Init opens a ZFS handle
func Init(nodePath string) error {
	if nodePath == "" {
		nodePath = "/dev/zfs"
	}
	var err error
	zfsHandle, err = os.Open(nodePath)
	if os.IsNotExist(err) {
		unix.Mknod(nodePath, 666, int(unix.Mkdev(10, 54)))
	}
	zfsHandle, err = os.Open(nodePath)
	if err != nil {
		return fmt.Errorf("Failed to open or create ZFS device node: %v", err)
	}
	return nil
}

type VDev struct {
	IsLog                uint64 `nvlist:"is_log"`
	SpaceMapObjectNumber uint64 `nvlist:"DTL,omitempty"`
	AlignmentShift       uint64 `nvlist:"ashift,omitempty"`
	AllocatableCapacity  uint64 `nvlist:"asize,omitempty"`
	GUID                 uint64 `nvlist:"guid,omitempty"`
	ID                   uint64 `nvlist:"id,omitempty"`
	Path                 string `nvlist:"path"`
	Type                 string `nvlist:"type"`
	Children             []VDev `nvlist:"children,omitempty"`
	L2CacheChildren      []VDev `nvlist:"l2cache,omitempty"`
	SparesChildren       []VDev `nvlist:"spares,omitempty"`
}

type PoolConfig struct {
	NumberOfChildren uint64 `nvlist:"vdev_children"`
	VDevTree         *VDev  `nvlist:"vdev_tree"`
	Errata           uint64 `nvlist:"errata,omitempty"`
	HostID           uint64 `nvlist:"hostid,omitempty"`
	Hostname         string `nvlist:"hostname,omitempty"`
	Name             string `nvlist:"name,omitempty"`
	GUID             uint64 `nvlist:"pool_guid,omitempty"`
	State            uint64 `nvlist:"state,omitempty"`
	TXG              uint64 `nvlist:"txg,omitempty"`
	Version          uint64 `nvlist:"version,omitempty"`
}

func delimitedBufToString(buf []byte) string {
	i := 0
	for ; i < len(buf); i++ {
		if buf[i] == 0x00 {
			break
		}
	}
	return string(buf[:i])
}

func stringToDelimitedBuf(str string, buf []byte) error {
	if len(str) > len(buf)-1 {
		return fmt.Errorf("String longer than target buffer (%v > %v)", len(str), len(buf)-1)
	}
	for i := 0; i < len(str); i++ {
		if str[i] == 0x00 {
			return errors.New("String contains null byte, this is unsupported by ZFS")
		}
		buf[i] = str[i]
	}
	return nil
}

// DatasetListNext lists ZFS datsets under the dataset or zpool given by name. It only returns one dataset and
// a cursor which can be used to get the next dataset in the list. The cursor value for the first element is 0.
func DatasetListNext(name string, cursor uint64, props interface{}) (string, uint64, DMUObjectSetStats, error) {
	cmd := &Cmd{
		Cookie: cursor,
	}
	if err := NvlistIoctl(zfsHandle.Fd(), ZFS_IOC_DATASET_LIST_NEXT, name, cmd, nil, props, nil); err != nil {
		return "", 0, DMUObjectSetStats{}, err
	}
	return delimitedBufToString(cmd.Name[:]), cmd.Cookie, cmd.Objset_stats, nil
}

// PoolCreate creates a new zpool with the given name, featues and devices
func PoolCreate(name string, features map[string]uint64, config VDev) error {
	cmd := &Cmd{}
	return NvlistIoctl(zfsHandle.Fd(), ZFS_IOC_POOL_CREATE, name, cmd, features, nil, config)
}

// PoolDestroy removes a zpool completely
func PoolDestroy(name string) error {
	cmd := &Cmd{}
	return NvlistIoctl(zfsHandle.Fd(), ZFS_IOC_POOL_DESTROY, name, cmd, nil, nil, nil)
}

// Promote replaces a ZFS filesystem with a clone of itself.
func Promote(name string) (conflictingSnapshot string, err error) {
	cmd := &Cmd{}
	err = NvlistIoctl(zfsHandle.Fd(), ZFS_IOC_PROMOTE, name, cmd, nil, nil, nil)
	conflictingSnapshot = delimitedBufToString(cmd.String[:])
	return
}

// Clone creates a new writable ZFS dataset from the given origin snapshot
func Clone(origin string, name string, props *DatasetProps) error {
	var cloneReq struct {
		Origin string        `nvlist:"origin"`
		Props  *DatasetProps `nvlist:"props"`
	}
	cloneReq.Origin = origin
	cloneReq.Props = props
	errList := make(map[string]int32)
	cmd := &Cmd{}
	return NvlistIoctl(zfsHandle.Fd(), ZFS_IOC_CLONE, name, cmd, cloneReq, errList, nil)
	// TODO: Partial failures using errList
}

// Create creates a new ZFS dataset
func Create(name string, t ObjectType, props *DatasetProps) error {
	var createReq struct {
		Type  ObjectType    `nvlist:"type"`
		Props *DatasetProps `nvlist:"props"`
	}
	createReq.Type = t
	createReq.Props = props
	cmd := &Cmd{}
	createRes := make(map[string]int32)
	return NvlistIoctl(zfsHandle.Fd(), ZFS_IOC_CREATE, name, cmd, createReq, createRes, nil)
}

// Snapshot creates one or more snapshots of datasets on the same zpool. The names are in standard
// ZFS syntax (dataset/subdataset@snapname).
func Snapshot(names []string, pool string, props *DatasetProps) error {
	var snapReq struct {
		Snaps map[string]bool `nvlist:"snaps"`
		Props *DatasetProps   `nvlist:"props"`
	}
	snapReq.Snaps = make(map[string]bool)
	for _, name := range names {
		if _, ok := snapReq.Snaps[name]; ok {
			return errors.New("duplicate snapshot name")
		}
		snapReq.Snaps[name] = true
	}
	snapReq.Props = props
	cmd := &Cmd{}
	snapRes := make(map[string]int32)
	return NvlistIoctl(zfsHandle.Fd(), ZFS_IOC_SNAPSHOT, pool, cmd, snapReq, snapRes, nil)
	// TODO: Maybe there is an error in snapRes
}

// DestroySnapshots removes multiple snapshots in the same pool. By setting the defer option the
// operation will be executed in the background after the function has returned.
func DestroySnapshots(names []string, pool string, defer_ bool) error {
	var destroySnapReq struct {
		Snaps map[string]bool `nvlist:"snaps"`
		Defer bool            `nvlist:"defer"`
	}
	destroySnapReq.Snaps = make(map[string]bool)
	for _, name := range names {
		if _, ok := destroySnapReq.Snaps[name]; ok {
			return errors.New("duplicate snapshot name")
		}
		destroySnapReq.Snaps[name] = true
	}
	destroySnapReq.Defer = defer_
	errList := make(map[string]int32)
	cmd := &Cmd{}
	return NvlistIoctl(zfsHandle.Fd(), ZFS_IOC_CLONE, pool, cmd, destroySnapReq, errList, nil)
}

// Bookmark creates ZFS bookmarks from snapshots. These are only available on ZoL 0.7+ and currently
// only used for resumable send/receive, but will eventually be usable as a reference for incremental
// sends.
func Bookmark(snapshotsToBookmarks map[string]string) error {
	errList := make(map[string]int32)
	cmd := &Cmd{}
	return NvlistIoctl(zfsHandle.Fd(), ZFS_IOC_BOOKMARK, "", cmd, snapshotsToBookmarks, errList, nil)
	// TODO: Handle errList
}

// Rollback rolls back a ZFS dataset to a snapshot taken earlier
func Rollback(name string, target string) (actualTarget string, err error) {
	var req struct {
		Target string `nvlist:"target,omitempty"`
	}
	req.Target = target
	var res struct {
		Target string `nvlist:"target"`
	}
	cmd := &Cmd{}
	err = NvlistIoctl(zfsHandle.Fd(), ZFS_IOC_ROLLBACK, name, cmd, req, res, nil)
	actualTarget = res.Target
	return
}

// PropSource represents all possible sources for ZFS props
type PropSource uint64

// All possible values of PropSource
const (
	PropSourceNone PropSource = 1 << iota
	PropSourceDefault
	PropSourceTemporary
	PropSourceLocal
	PropSourceInherited
	PropSourceReceived
)

// SetProp sets one or more props on a ZFS dataset.
func SetProp(name string, props map[string]interface{}, source PropSource) error {
	cmd := &Cmd{
		Cookie: uint64(source),
	}
	errList := make(map[string]int64)
	return NvlistIoctl(zfsHandle.Fd(), ZFS_IOC_SET_PROP, name, cmd, props, errList, nil)
	// TODO: Distinguish between partial and complete failures using errList
}

// InheritProp makes a prop inherit from its parent or reverts it to the received prop which is
// being shadowed by a local prop (see PropSource).
func InheritProp(name string, propName string, revertToReceived bool) error {
	var cookie uint64
	if revertToReceived {
		cookie = 1
	}
	cmd := &Cmd{
		Cookie: cookie,
	}
	if err := stringToDelimitedBuf(propName, cmd.Value[:]); err != nil {
		return err
	}
	return NvlistIoctl(zfsHandle.Fd(), ZFS_IOC_INHERIT_PROP, name, cmd, nil, nil, nil)
}

// GetSpaceWritten returns the amount of bytes written into a dataset since the given snapshot was
// taken. Also useful for determining if anything has changed in dataset since the snaphsot was taken.
func GetSpaceWritten(dataset, snapshot string) (uint64, error) {
	cmd := &Cmd{}
	stringToDelimitedBuf(snapshot, cmd.Value[:])
	if err := NvlistIoctl(zfsHandle.Fd(), ZFS_IOC_SPACE_WRITTEN, dataset, cmd, nil, nil, nil); err != nil {
		return 0, err
	}
	return cmd.Cookie, nil
}

// Rename renames a dataset
func Rename(oldName, newName string, recursive bool) error {
	var cookieVal uint64
	if recursive {
		cookieVal = 1
	}
	cmd := &Cmd{
		Cookie: cookieVal,
	}
	stringToDelimitedBuf(newName, cmd.Value[:])
	return NvlistIoctl(zfsHandle.Fd(), ZFS_IOC_RENAME, oldName, cmd, nil, nil, nil)
}

// Destroy removes dataset irrevocably. If the deferred flag is given, the function will terminate
// and the actuall removal will be processed asynchronously.
func Destroy(name string, t ObjectType, deferred bool) error {
	cmd := &Cmd{
		Objset_type: uint64(t),
	}
	return NvlistIoctl(zfsHandle.Fd(), ZFS_IOC_DESTROY, name, cmd, nil, nil, nil)
}

// SendSpaceOptions contains all options for the SendSpace function
type SendSpaceOptions struct {
	// From can contain an older snapshot for an incremental transfer
	From string `nvlist:"from,omitempty"`
	// These enable individual features for transfer space estimation
	LargeBlocks bool `nvlist:"largeblockok"`
	Embed       bool `nvlist:"embedok"`
	Compress    bool `nvlist:"compressok"`
}

// SendSpace determines approximately how big a ZFS send stream will be
func SendSpace(name string, options SendSpaceOptions) (uint64, error) {
	cmd := &Cmd{}
	var spaceRes struct {
		Space uint64 `nvlist:"space"`
	}
	if err := NvlistIoctl(zfsHandle.Fd(), ZFS_IOC_SEND_SPACE, name, cmd, options, &spaceRes, nil); err != nil {
		return 0, err
	}
	return spaceRes.Space, nil
}

type sendStream struct {
	peekBuf   []byte
	errorChan chan error
	lastError error
	isEOF     bool
	r         io.ReadCloser
}

func (s *sendStream) Read(buf []byte) (int, error) {
	if s.isEOF {
		return 0, s.lastError
	}
	if len(s.peekBuf) > 0 {
		n := copy(buf, s.peekBuf)
		s.peekBuf = s.peekBuf[n:]
		return n, nil
	}
	n, err := s.r.Read(buf)
	if err == io.EOF {
		s.lastError = <-s.errorChan
		if s.lastError == nil {
			s.lastError = io.EOF
		}
		s.isEOF = true
		return n, s.lastError
	}
	return n, err
}

func (s *sendStream) peek(buf []byte) (int, error) {
	if s.isEOF {
		return 0, s.lastError
	}
	n, err := s.r.Read(buf)
	s.peekBuf = append(s.peekBuf, buf[:n]...)
	if err == io.EOF {
		s.lastError = <-s.errorChan
		if s.lastError == nil {
			s.lastError = io.EOF
		}
		s.isEOF = true
		return n, s.lastError
	}
	return n, err
}

func (s sendStream) Close() error {
	return s.r.Close()
}

// SendOptions contains all options for the Send function.
type SendOptions struct {
	// Fd is writable file descriptor and should generally not be set. If it is set, all convenience
	// wrappers will be disabled and the Fd will be directly passed into the kernel.
	Fd int32 `nvlist:"fd"`

	// From can optionally contain an older snapshot for an incremental send
	From string `nvlist:"fromsnap,omitempty"`

	// These enable individual features for the send stream
	LargeBlocks bool `nvlist:"largeblockok"`
	Embed       bool `nvlist:"embedok"`
	Compress    bool `nvlist:"compress"`

	// These can optionally be set to resume a transfer (ZoL 0.7+)
	ResumeObject uint64 `nvlist:"resume_object,omitempty"`
	ResumeOffset uint64 `nvlist:"resume_offset,omitempty"`
}

// Send generates a stream containing either a full or an incremental snapshot. This function provides
// some basic convenience wrappers including a fail-fast mode which returns an error directly if it
// happens before a single byte is sent out and a Read-compatible output stream.
func Send(name string, options SendOptions) (io.ReadCloser, error) {
	cmd := &Cmd{}

	r, w, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	options.Fd = int32(w.Fd())

	stream := sendStream{
		errorChan: make(chan error, 1),
		r:         r,
	}

	go func() {
		err := NvlistIoctl(zfsHandle.Fd(), ZFS_IOC_SEND_NEW, name, cmd, options, &struct{}{}, nil)
		stream.errorChan <- err
		w.Close()
	}()

	buf := make([]byte, 1) // We want at least 1 byte of output to enter streaming mode

	_, err = stream.peek(buf)
	if err != nil {
		r.Close()
		w.Close()
		return nil, err
	}

	return &stream, nil
}

// PoolGetProps gets all props for a zpool
func PoolGetProps(name string) (props interface{}, err error) {
	props = new(interface{})
	cmd := &Cmd{}
	err = NvlistIoctl(zfsHandle.Fd(), ZFS_IOC_POOL_GET_PROPS, name, cmd, nil, props, nil)
	return
}

// ObjsetZPLProps gets all object set props
func ObjsetZPLProps(name string) (props interface{}, err error) {
	props = new(interface{})
	cmd := &Cmd{}
	if err = NvlistIoctl(zfsHandle.Fd(), ZFS_IOC_OBJSET_ZPLPROPS, name, cmd, nil, props, nil); err != nil {
		return
	}
	return
}

// ObjsetStats gets statistics on object sets
func ObjsetStats(name string) (props interface{}, err error) {
	props = new(interface{})
	cmd := &Cmd{}
	if err = NvlistIoctl(zfsHandle.Fd(), ZFS_IOC_OBJSET_STATS, name, cmd, nil, props, nil); err != nil {
		return
	}
	return
}
