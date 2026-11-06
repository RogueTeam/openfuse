= Writer and reader at once
    file readme 644 hello world
    open readme w1 O_RDWR|O_TRUNC 0
    writeall w1 new contents

    open readme r1 O_RDONLY 0
    readall r1 new contents

    close w1
    close r1
