= List twice
# - Create at root
file readme1 640 contents1
file readme2 640 contents2
# - Create a subdirectory
mkdir dir 700
file dir/readme1 640 contents1
file dir/readme2 640 contents2

# Do the listings
ls . dir readme1 readme2
ls . dir readme1 readme2
# List directory
ls dir readme1 readme2
ls dir readme1 readme2

= List, create, list
# - Create at root
file readme1 640 contents1
file readme2 640 contents2
# - Create a subdirectory
mkdir dir 700
file dir/readme1 640 contents1
file dir/readme2 640 contents2

# Do the listings
ls . dir readme1 readme2
file readme3 644 contents
ls . dir readme1 readme2 readme3

# List directory
ls dir readme1 readme2
file dir/readme3 644 contents
ls dir readme1 readme2 readme3


= List, delete, list
# - Create at root
file readme1 640 contents1
file readme2 640 contents2
# - Create a subdirectory
mkdir dir 700
file dir/readme1 640 contents1
file dir/readme2 640 contents2

# Do the listings
ls . dir readme1 readme2
rm readme2
ls . dir readme1

# List directory
ls dir readme1 readme2
rm dir/readme2
ls dir readme1

= Mid directory (500 entries)
mkdir mid 744
touchn mid/autogen- 700 500
lscount mid 500
rmcount mid 250
lscount mid 250

= Big directory (2000 entries)
mkdir mid 744
touchn mid/autogen- 700 2000
lscount mid 2000
rmcount mid 1999
lscount mid 1

= Deep nesting
mkdir a 700
mkdir a/b 700
mkdir a/b/c 700
mkdir a/b/c/d 700
file  a/b/c/d/readme 700 contents

lscount a 1
lscount a/b 1
lscount a/b/c 1
lscount a/b/c/d 1
