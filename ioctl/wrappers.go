package ioctl

import (
	"errors"
	"io"
	"os"
)

// DatasetProps contains all normal props for a dataset
type DatasetProps struct {
}

// PoolProps represents all properties of a zpool
type PoolProps struct {
	Name    string `nvlist:"name"`
	Version uint64 `nvlist:"version"`
	Comment string `nvlist:"comment"`

	// Pool configuration
	AlternativeRoot string   `nvlist:"altroot"`
	TemporaryName   string   `nvlist:"tname"`
	BootFS          string   `nvlist:"bootfs"`
	CacheFile       string   `nvlist:"cachefile"`
	ReadOnly        bool     `nvlist:"readonly"`
	Multihost       bool     `nvlist:"multihost"`
	Failmode        FailMode `nvlist:"failmode"`
	DedupDitto      uint64   `nvlist:"dedupditto"`
	AlignmentShift  uint64   `nvlist:"ashift"`
	Delegation      bool     `nvlist:"delegation"`
	Autoreplace     bool     `nvlist:"autoreplace"`
	ListSnapshots   bool     `nvlist:"listsnapshots"`
	Autoexpand      bool     `nvlist:"autoexpand"`
	MaxBlockSize    uint64   `nvlist:"maxblocksize"`
	MaxDnodeSize    uint64   `nvlist:"maxdnodesize"`

	// Defines props for the root volume for PoolCreate()
	RootProps *DatasetProps `nvlist:"root-props-nvl"`

	// All user properties are represented here
	User map[string]string `nvlist:"-,extra"`

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

func init() {
	zfsHandle, _ = os.Open("/dev/zfs")
}

type Child struct {
	IsLog bool `nvlist:"is_log"`
}

type Disk struct {
	SpaceMapObjectNumber uint64 `nvlist:"DTL"`
	AlignmentShift       uint64 `nvlist:"ashift"`
	AllocatableCapacity  uint64 `nvlist:"asize"`
}

type PoolConfig struct {
	Children []Child `nvlist:"children"`
	Spares   []Child `nvlist:"spares"`
	L2Cache  []Child `nvlist:"l2cache"`
}

/*func DatasetListNext(name string, cookie uint64) (string, uint64, DMUObjectSetStats, DatasetProps, error) {

}*/

func PoolCreate(name string, options PoolProps, config PoolConfig) error {
	return nil
}

func Create(name string, t ObjectType, props *DatasetProps) error {
	var createReq struct {
		Type  ObjectType    `nvlist:"type"`
		Props *DatasetProps `nvlist:"props"`
	}
	createReq.Type = t
	createReq.Props = props
	cmd := &Cmd{}
	createRes := make(map[string]int32)
	return NvlistIoctl(zfsHandle.Fd(), ZFS_IOC_CREATE, name, cmd, createReq, createRes)
}

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
	return NvlistIoctl(zfsHandle.Fd(), ZFS_IOC_SNAPSHOT, pool, cmd, snapReq, snapRes)
	// TODO: Maybe there is an error in snapRes
}

func GetWrittenProperty(dataset, snapshot string) (uint64, error) {
	cmd := &Cmd{}
	if len(snapshot) > 8191 {
		return 0, errors.New("snapshot is longer than 8191 bytes, this is unsupported")
	}
	for i := 0; i < len(snapshot); i++ {
		cmd.Value[i] = snapshot[i]
	}
	if err := NvlistIoctl(zfsHandle.Fd(), ZFS_IOC_SPACE_WRITTEN, dataset, cmd, nil, nil); err != nil {
		return 0, err
	}
	return cmd.Cookie, nil
}

func Rename(oldName, newName string, recursive bool) error {
	var cookieVal uint64
	if recursive {
		cookieVal = 1
	}
	cmd := &Cmd{
		Cookie: cookieVal,
	}
	if len(newName) > 8191 {
		return errors.New("newName is longer than 8191 bytes, this is unsupported")
	}
	for i := 0; i < len(newName); i++ {
		cmd.Value[i] = newName[i]
	}
	return NvlistIoctl(zfsHandle.Fd(), ZFS_IOC_RENAME, oldName, cmd, nil, nil)
}

func Destroy(name string, t ObjectType, deferred bool) error {
	cmd := &Cmd{
		Objset_type: uint64(t),
	}
	return NvlistIoctl(zfsHandle.Fd(), ZFS_IOC_DESTROY, name, cmd, nil, nil)
}

type SendSpaceOptions struct {
	From        string `nvlist:"from,omitempty"`
	LargeBlocks bool   `nvlist:"largeblockok"`
	Embed       bool   `nvlist:"embedok"`
	Compress    bool   `nvlist:"compress"`
}

func SendSpace(name string, options SendSpaceOptions) (uint64, error) {
	cmd := &Cmd{}
	var spaceRes struct {
		Space uint64 `nvlist:"space"`
	}
	if err := NvlistIoctl(zfsHandle.Fd(), ZFS_IOC_SEND_SPACE, name, cmd, options, &spaceRes); err != nil {
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

type SendOptions struct {
	Fd           int32  `nvlist:"fd"`
	From         string `nvlist:"fromsnap,omitempty"`
	LargeBlocks  bool   `nvlist:"largeblockok"`
	Embed        bool   `nvlist:"embedok"`
	Compress     bool   `nvlist:"compress"`
	ResumeObject uint64 `nvlist:"resume_object,omitempty"`
	ResumeOffset uint64 `nvlist:"resume_offset,omitempty"`
}

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
		err := NvlistIoctl(zfsHandle.Fd(), ZFS_IOC_SEND_NEW, name, cmd, options, &struct{}{})
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

func PoolGetProps(name string) (props interface{}, err error) {
	props = new(interface{})
	cmd := &Cmd{}
	err = NvlistIoctl(zfsHandle.Fd(), ZFS_IOC_POOL_GET_PROPS, name, cmd, nil, props)
	return
}

func ObjsetZPLProps(name string) (props interface{}, err error) {
	props = new(interface{})
	cmd := &Cmd{}
	if err = NvlistIoctl(zfsHandle.Fd(), ZFS_IOC_OBJSET_ZPLPROPS, name, cmd, nil, props); err != nil {
		return
	}
	return
}

func ObjsetStats(name string) (props interface{}, err error) {
	props = new(interface{})
	cmd := &Cmd{}
	if err = NvlistIoctl(zfsHandle.Fd(), ZFS_IOC_OBJSET_STATS, name, cmd, nil, props); err != nil {
		return
	}
	return
}
