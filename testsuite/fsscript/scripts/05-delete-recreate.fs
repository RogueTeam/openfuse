= Delete, then recreate the same name
    file readme 600 one
    rm readme
    file readme 644 two
    cat readme two
    perm readme 644

= Delete an Open File
    file readme 600 hello world
    open readme h1 O_RDONLY 0
    rm readme
    readall h1 hello world
    close h1

= Delete while open, then recreate
    file readme 600 hello world
    open readme h1 O_RDONLY 0
    rm readme
    file readme 644 new contents
    readall h1 hello world
    close h1
    cat readme new contents
