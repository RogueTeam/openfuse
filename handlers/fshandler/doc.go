// Helper handler implemented for two main reason
// Reason 1: to have a reference to other library users on how to implement custom handlers
// to pass to the library.
// Reason 2: to have a helper handler that translates a fs.FS into a real unix filesystem.
//
// Important reads:
// fshandler.FullFS: Any filesystem implementing the methods defined in this type will support a full RW filesystem
// fshandler.FullDir | fshandler.FullFile: Any file and directory implementing these interface will support full file operations
//
// Go stdlib filesystem are first class citizens, meaning embed.FS and os.DirFS work outof the box for read-only filesystems
// For custom implementations it is recommended to actually built your own so you can have thinks like synthetic files.
//
// # A unmapped type that is also interesting to implement is the ExtFileInfo when returning any kind of FileInfo, this permits granular control over more fields of the return attrs
//
// Check memfs package as an example reference
package fshandler
