#!/usr/bin/env python
"""mapper.py"""

import sys
import re

# Gets the data from Standard Input
for line in sys.stdin:
    # Make a Regex to say we only want letters, space and numbers
    line = re.sub('[^a-zA-Z0-9 ]+', ' ', line.strip())
    for word in line.split(" "):
        if word == "": #Ignore if empty string
            continue
        print(word + "\t1") #Send the output to standard output
