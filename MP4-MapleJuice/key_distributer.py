import os

ip_list = ["fa19-cs425-g69-01.cs.illinois.edu",
    "fa19-cs425-g69-02.cs.illinois.edu",
    "fa19-cs425-g69-03.cs.illinois.edu",
    "fa19-cs425-g69-04.cs.illinois.edu",
    "fa19-cs425-g69-05.cs.illinois.edu",
    "fa19-cs425-g69-06.cs.illinois.edu",
    "fa19-cs425-g69-07.cs.illinois.edu",
    "fa19-cs425-g69-08.cs.illinois.edu",
    "fa19-cs425-g69-09.cs.illinois.edu",
    "fa19-cs425-g69-10.cs.illinois.edu"]

for ip in ip_list:
    command = 'cat .ssh/id_rsa.pub | ssh pihess@{0} "cat >> ~/.ssh/authorized_keys"'.format(ip)
    os.system(command)





