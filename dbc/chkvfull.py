#!/usr/bin/env python3

import time
try:
	import requests
except ImportError as e:
	print("Error importing requests. Is it installed?")
	exit()

def main():
	verify_list = []
	verify_file = open('verify.urls', 'r')
	for line in verify_file:
		verify_list.append(line.strip('\n'))
	verify_file.close()
	new_verify_file = open('n_verify.urls', 'w')
	for item in verify_list:
		try:
			ret = requests.get(item)
			if ('Thank you!' in ret.text):
				verify_list.remove(item)
			else:
				# why remove the items if successful if we just write em here? hmm...
				newnew = item + '\n'
				new_verify_file.write(newnew)
		except IOError as e:
			elog = open("error.log", "a+")
			elog.write('chkv(exception) => ' + e + '::' + ret.text + '\n')
			elog.close()
			# sleep and attempt to retry... todo: make failsafe if continues to ioerror
			time.sleep(10)
			ret = requests.get(item)
			if ('Thank you!' in ret.text):
				verify_list.remove(item)
			else:
				# why remove the items if successful if we just write em here? hmm...
				newnew = item + '\n'
				new_verify_file.write(newnew)
		time.sleep(2.1) # we get limited if we don't sleep :{}

if __name__ == '__main__':
	main()
