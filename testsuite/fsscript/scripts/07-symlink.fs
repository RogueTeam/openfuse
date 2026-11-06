= Create and read a symlink
    file readme 644 hello world
    symlink readme readme.link
    readlink readme.link readme
    cat readme.link hello world

    rm readme
    ! cat readme.link hello world

= Dangling symlink
    file readme 644 hello world
    symlink readme readme.link
    readlink readme.link readme
    cat readme.link hello world

    rm readme
    ! cat readme.link hello world

= Symlink to directory
    mkdir dir 700
    file dir/readme 644 hello world

    symlink dir dir.link
    readlink dir.link dir

    cat dir.link/readme hello world

= Delete symlink
    file readme 644 hello world
    symlink readme readme.link
    readlink readme.link readme
    cat readme.link hello world

    rm readme.link
    ! cat readme.link hello world
    cat readme hello world

= Rename a link
    file readme 644 hello world
    symlink readme readme.link
    readlink readme.link readme
    cat readme.link hello world

    mv readme.link new.link
    ! cat readme.link hello world
    cat new.link hello world
