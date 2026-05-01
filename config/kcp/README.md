# KCP Support

The controllers are based on the `kubecrtutls` library, which provides
support for KCP.

The example manifests define an API export and appropriate bindings.
If the replication or server mode is used, they can be simplified by omitting
the requested additional resources, required to grant permissions to
access the resources in a particular namespace and workspace.