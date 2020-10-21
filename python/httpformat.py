import sys, string
#read for proxy files; won't work if terminal isn't cd'd into the project dir
formatted_proxy_file = open("list.proxies", "wb")
with open("proxy_http_ip.txt", "r") as proxy_file:
	for line in proxy_file:
		formatted_proxy_file.write(bytes("http://"+line.strip("\n")+"\n", "utf-8"))
print("Finished! Formatted proxies saved to list.proxies")
