= Writer and reader at once
    file readme 644 hello world
    open readme w1 O_RDWR|O_TRUNC 0
    writeall w1 new contents

    open readme r1 O_RDONLY 0
    readall r1 new contents

    close w1
    close r1

= Stat while writing
    file readme 644 hello world
    open readme w1 O_RDWR|O_TRUNC 0
    writeall w1 new contents

    size readme 12

    cat readme new contents
    close w1
    cat readme new contents

= fsync, overwrite, close
    open readme w1 O_RDWR|O_CREATE|O_TRUNC 644
    writeall w1 hello
    fsync w1
    seek w1 0 0
    writeall w1 HELLO
    close w1
    cat readme HELLO
