import sys, string
inc = 25
amt = 0
#read for proxy files; won't work if terminal isn't cd'd into the project dir
with open("proxy_socks_ip.txt", "r") as proxy_file:
	for line in proxy_file:
		ud = line.strip("\n")
		ux = ud.index(":")
		amt += 1
		print(ud[:ux])
		if amt == inc:
			amt = 0
			print("Enter to continue...")
			input()
print("Finished!")
