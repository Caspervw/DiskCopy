# DiskCopy
Makes sure that everything from disk A ends up on disk B

One of my hard drives is failing and my backup is not in sync. I don't currently know what the state is, so I want everything from disk A to be moved to disk B. The following rules will apply:

If the file on disk A is **somewhere** on disk B, than I'm happy.
If the file is not on disk B I want it there, but since the structure on disk B might have changed, we're going to put it in a separate folder (“to archive”-folder) on disk B.

Result: The contents on disk A are copied to B, but you have to manually make sure that you archive the files in the “to archive-folder. Hopefully it’s just a couple of files :)
