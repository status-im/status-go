## The command line std/cmdline reads. liblogosdelivery builds with --noMain,
## which leaves it undefined, and the seal keeps the unused readers in the one
## object a program links.
{.emit: """
N_LIB_PRIVATE int cmdCount;
N_LIB_PRIVATE char** cmdLine;
""".}
