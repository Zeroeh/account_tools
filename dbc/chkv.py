#!/usr/bin/env python3

import time
try:
	import requests
except ImportError as e:
	print("Error importing requests. Is it installed?")
	exit()

def finishWrite(acclist, nvfile):
	for item in acclist:
		newnew = item + '\n'
		nvfile.write(newnew)
	nvfile.close()
	exit(0)

def main():
	verify_list = []
	noverifycount = 0
	counter = 0
	verify_file = open('verify.urls', 'r')
	for line in verify_file:
		verify_list.append(line.strip('\n'))
	verify_file.close()
	new_verify_file = open('n_verify.urls', 'w')
	for item in verify_list:
		counter += 1
		if noverifycount > 30:
			finishWrite(verify_list, new_verify_file)
		try:
			ret = requests.get(item)
			if ('Thank you' in ret.text):
				verify_list.remove(item)
				print("Removed account")
				noverifycount = 0
			else:
				noverifycount += 1
		except IOError as e:
			elog = open("error.log", "a+")
			elog.write('chkv(exception) => ' + e + '::' + ret.text + '\n')
			elog.close()
			time.sleep(10)
			ret = requests.get(item)
			if ('Thank you' in ret.text):
				verify_list.remove(item)
				print("Removed account")
				noverifycount = 0
			else:
				noverifycount += 1
		time.sleep(2.1)

if __name__ == '__main__':
	main()
