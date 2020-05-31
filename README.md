# GoZFS
*A pure-Go minimalist ZFS userspace implementation for storage systems*

## Design goals
* Ease of use
* Fine-grained control, nothing is automatically done
* Performance
* Compatibility with multiple ZoL releases

## Differences from normal ZFS userspace
GoZFS is only a wrapper around the kernel interface of ZoL. It won't automount/auto-unmount anything,
it disregards all props (for example `canmount`), it doesn't use ZFS history and much more.
This has certain disadvantages when writing a ZFS command-line utility that people expect to work
exactly like a normal ZFS would, but is hugely benefical to storage systems written on top of ZFS,
since they have very tight control of what is actually done.
GoZFS is designed to be compatible with ZoL 0.6+ in a single library,
it does not need to be kept in lock-step with the ZoL in-kernel module.
Because of this and its intended use case in other software its own
error handling is very minimal and generally relies only on the kernel.

## Architecture
GoZFS currently consists of 2 packages and will eventually consist of three:
* `ioctl`: Slim wrappers around the pure ZFS ioctls, these only do the bare minimum to make the
  ioctls usable and memory-safe. Covers most relevant ioctls now.
* `nvlist`: A pure-Go implementation of ZFS's flavor of nvlists with a similar API to `encoding/json`.
   Only implements the bits necessary for ZFS. Mostly for internal use by the `ioctl` package, but
   not tied to it.
* `zfs`: A wrapper around the `ioctl` package to make the API more Go-like and convenient to use.
  Not yet implemented.

## Utilities
GoZFS provides a custom strace implementation for tracing ZFS ioctls. It is found under `ioctl/trace`
and can be used to inspect calls by GoZFS or the normal ZFS userspace utilities.

## Stability & Testing
This is currently alpha-level software. Its implementation and API is still incomplete and subject to change.
It does work for most standard storage system tasks, but there is minimal documentation. The high-levl interface
package, `zfs` still remains to be written.

There is automated integration testing against a custom 4.19 kernel with ZoL 0.8 inside
kvmtool in GitLab CI/Kubernetes. The tests can also be run standalone if you have a working ZoL setup.
Full matrix testing against ZoL 0.7 on Linux 4.19 and ZoL 0.6 on Linux 4.9 is planned. The test runtime
cannot be distributed since it contains compiled CDDL and GPLv2 code.

The decoder side of nvlist has a fuzzing harness based on go-fuzz.

## Not yet implemented
* Import from on-disk labels (missing proper XDR support in nvlist)
* VDev management (needs reverse-engineered config structures)
* Feature management (upgrade, enabling, disabling)
* Diff
* Encryption
* Import cache

## Out-of-scope
* History
* Fault injection
* ACLs (currently don't see where it might be reasonably used, but I'm open to a discussion)
* SMB/NFS sharing (do it inside your storage system)

