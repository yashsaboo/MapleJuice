#!/usr/bin/env python
"""reverseWebLinkReduce.py"""

import sys

d = dict() #Create dcitionary for storing key,values
# Gets the data from STDIN
for line in sys.stdin:
    pageB, pageA = line.strip().split('\t') #Get the pageB,pageA
    d[pageB] = d.get(pageB, "") + " " + pageA #Append the pageA
    
for key,val in sorted(d.items()): 
    print(key + "\t" + str(val.strip())) #Send the output to STDOUT
