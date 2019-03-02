# Clean up PVCs whose namespace has been deleted

This script will delete PVCs that are stuck in the terminating
state due to their namespaces having been fully deleted without
them being cleaned up.

The procedure is as follows:

1. Re-create their namespace
1. Delete all deployments and stateful sets that might be using the PVC
1. Delete the PVC
1. Delete the namespace again