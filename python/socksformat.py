import sys, string
#read for proxy files; won't work if terminal isn't cd'd into the project dir
formatted_proxy_file = open("socks.proxies", "wb")
with open("proxy_socks_ip.txt", "r") as proxy_file:
	for line in proxy_file:
		formatted_proxy_file.write(bytes("socks5://"+line.strip("\n")+"\n", "utf-8"))
print("Finished! Formatted proxies saved to socks.proxies")

#note: the socks5:// is intended as the golang proxy updater / encoder.go will remove it
