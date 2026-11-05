= Create with several permissions
    file readme1 600 EMPTY
    file readme2 640 EMPTY
    file readme3 755 EMPTY
    perm readme1 600
    perm readme2 640
    perm readme3 755

= chmod file
    file readme 600 EMPTY
    perm readme 600
    chmod readme 644
    perm readme 644

= chmod directory
    mkdir secrets 600
    perm secrets 600
    chmod secrets 644
    perm secrets 644

= chmod, then read
    file readme 600 contents
    perm readme 600
    cat readme contents
    chmod readme 644
    perm readme 644
    cat readme contents

= chmod twice
    file readme 600 contents
    perm readme 600
    cat readme contents
    chmod readme 644
    perm readme 644
    cat readme contents
    chmod readme 777
    perm readme 777
    cat readme contents

= Create mode isn't affect by a later chmod of the parent
    mkdir parent 700
    file parent/readme 640 contents
    perm parent 700
    perm parent/readme 640
    cat parent/readme contents

    chmod parent 666
    perm parent 666
    perm parent/readme 640
    cat parent/readme contents
