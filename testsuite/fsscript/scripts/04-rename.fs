= Rename in place
    file readme 700 hello world
    mv readme readme.bak
    ! cat readme hello world
    cat readme.bak hello world
    lscount . 1

= Move across directories, then rename the old one
    mkdir a 700
    mkdir b 700
    file a/f 644 hello world
    mv a/f b/f
    lscount a 0
    lscount b 1
    ! cat a/f hello world
    cat b/f hello world
    mv a a2
    lscount a2 0
    ! cat a2/f hello world

= Rename a directory with nested content
    mkdir a 700
    mkdir a/x 700
    file a/x/y 644 hello world
    cat a/x/y hello world
    mv a b
    ! cat a/x/y hello world
    cat b/x/y hello world

    lscount b/x 1

= Move into a directory, then rename that directory
    file readme 644 hello world
    mkdir dir 700
    mv readme dir/readme
    mv dir dir.bak
    cat dir.bak/readme hello world
    ! cat dir/readme hello world

= Editor style save
    file t 644 old
    file s 644 new
    mv s t
    cat t new
    ! cat s new

= Rename a directory into itself
    # Level 1
    mkdir a 700
    mkdir a/b 700
    mkdir a/b/c 700
    mkdir a/b/c/d 700
    mkdir a2 700
    mkdir a2/b 700

    ! mv a a/b
    ! mv a a/b/c
    ! mv a a/b/c/d
    mv a a2/b/a
    ls . a2
    ls a2/b a
    lscount a2/b/a/b/c/d 0


= Rename into a sibling whose name starts the same
    mkdir dir 700
    mkdir dir2 700
    mv dir dir2/x
    lscount . 1
    lscount dir2 1

= Rename onto itself
    file readme 644 hello world
    mv readme readme
    cat readme hello world

= Directory over an empty directory
    mkdir dir 700
    file dir/readme 644 hello world
    mkdir target 700
    ! mv dir target # This behavior is correct of OpenBSD. Tested with FFS
    cat dir/readme hello world

= Directory over file and file over directory
    mkdir dir 700
    file readme 644 hello world

    ! mv dir readme
    ! mv readme dir

    lscount dir 0
    cat readme hello world

= Rename while open
    file readme 644 hello world
    open readme h1 O_RDONLY 0
    mv readme readme2
    cat readme2 hello world
    readall h1 hello world
    close h1
    ! cat readme hello world
    cat readme2 hello world
