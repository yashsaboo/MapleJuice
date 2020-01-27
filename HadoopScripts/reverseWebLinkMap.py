#!/usr/bin/env python
"""reverseWebLinkMap.py"""

import sys

# Gets the data from STDIN
for line in sys.stdin:
    page = line.strip().split("\t") #Remove whitespaces and split it with tab character, since dataset uses tab as a delimeter
    print(page[1] + "\t" + page[0]) #Send the output to STDOUT