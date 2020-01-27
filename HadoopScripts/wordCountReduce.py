#!/usr/bin/env python
"""wordCountReduce.py"""

import sys

d = dict() #Create dcitionary for storing key,values
for line in sys.stdin:
    word, count = line.strip().split('\t')
    d[word] = d.get(word, 0) + 1
    
for key,val in d.items():
    print(key + "\t" + str(val))
