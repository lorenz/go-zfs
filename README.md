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
since they have very thight control of what is actually done.
GoZFS is designed to be compatible with ZoL 0.6+ in a single library,
it does not need to be kept in lock-step with the ZoL in-kernel module.
Because of this and its intended use case in other software its own
error handling is very minimal and generally relies only on the kernel.

## Not yet implemented
* Receive (missing Fletcher4-based byte-order detection)
* Import (missing proper XDR support in nvlist)
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

