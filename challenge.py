import struct
import sys

class SynChallenge:
	parser = struct.Struct('<H')
	collection = []
	index = 0
	register = [0, 0, 0, 0, 0, 0, 0, 0]
	stack = []

	def __init__(self):
		f = open('challenge.bin', 'rb')
		try:
			code = f.read(2)
			while (code != ''):
				self.collection.append(code)
				code = f.read(2)
		finally:
			f.close

	def isdebug(self):
		return (len(sys.argv) > 1 and sys.argv[1] == '-d');

	def debug(self, text):
		if (self.isdebug()):
			print text

	def setindex(self, index):
		self.index = index % 32768

	def readbyte(self, expand_register = 'yes'):
		byte = self.parser.unpack(self.collection[self.index])[0]
		if (byte < 32768 or expand_register == 'raw'):
			self.debug('reading index ' + str(self.index) + ', value ' + str(byte))
			if (expand_register is not 'yes' and expand_register is not 'raw'):
				self.debug('** expand_register was set to ' + str(expand_register) + ' but byte is not a register')
			return byte
		elif (byte < 32776):
			if (expand_register == 'yes'):
				self.debug('reading register ' + str(byte - 32768) + ', value ' + str(self.register[byte - 32768]))
				return self.register[byte - 32768]
			else:
				self.debug('reading register ' + str(byte - 32768) + ' and returning as the value')
				return byte - 32768 # but what if you want the full register
		else:
			sys.exit('BZZT ' + str(byte) + ' IS NOT VALID')

	def writebyte(self, index, value):
		if (index < 32768):
			self.debug('writing ' + str(value) + ' to byte ' + str(index))
			self.collection[index] = self.parser.pack(value)
		elif (index < 32776):
			index -= 32768
			self.setregister(index, value)

	def readnext(self, expand_register = 'yes'):
		self.setindex(self.index + 1)
		return self.readbyte(expand_register)

	def jmp(self, index):
		self.setindex(index)
		self.parse()

	def setregister(self, index, value):
		self.register[index] = value
		self.debug('setting register ' + str(index) + ' to ' + str(value) + ' (register is ' + str(self.register) + ')')

	def call(self):
		self.stack.append(self.index + 2)
		self.debug('call: writing ' + str(self.index + 2) + ' to the stack (stack is ' + str(self.stack) + ')')
		self.jmp(self.readnext())

	def parse(self):
		curr = self.readbyte()
		if (curr == 0): # halt
			sys.exit()
		elif (curr == 1): #set
			self.setregister(self.readnext(False), self.readnext())
		elif (curr == 2): # push
			value = self.readnext()
			self.stack.append(value)
			self.debug('appended ' + str(value) + ' to stack (stack is ' + str(self.stack) + ')')
		elif (curr == 3): #pop
			index = self.readnext('raw')
			value = self.stack.pop()
			self.writebyte(index, value)
			self.debug('popped ' + str(value) + ' off the stack and wrote it to byte ' + str(index) + ' (stack is ' + str(self.stack) + ')')
		elif (curr == 4): # eq
			index = self.readnext('raw')
			a = self.readnext()
			b = self.readnext()
			if (a == b):
				self.writebyte(index, 1)
			else:
				self.debug(str(a) + ' is not equal to ' + str(b))
				self.writebyte(index, 0)
		elif (curr == 5): # gt
			index = self.readnext('raw')
			if (self.readnext() > self.readnext()):
				self.writebyte(index, 1)
			else:
				self.writebyte(index, 0)
		elif (curr == 6): # jmp
			self.jmp(self.readnext())
		elif (curr == 7): # jt
			value = self.readnext()
			if (value != 0):
				self.debug(str(self.index) + ' is ' + str(value) + ', not 0, jmping')
				self.jmp(self.readnext())
			else:
				self.debug(str(self.index) + ' is 0, no jmp')
				self.setindex(self.index + 1)
		elif (curr == 8): # jf
			value = self.readnext()
			if (value == 0):
				self.debug(str(self.index) + ' is 0, jmping')
				self.jmp(self.readnext())
			else:
				self.debug(str(self.index) + ' is ' + str(value) + ', not 0, no jmp')
				self.index += 1
		elif (curr == 9): # add
			index = self.readnext('raw')
			value = (self.readnext() + self.readnext()) % 32768
			self.writebyte(index, value)
		elif (curr == 10): # mult
			index = self.readnext('raw')
			value = (self.readnext() * self.readnext()) % 32768
			# so, fun times: if you accidentally multiply three numbers above it skips to the next part without
			# actually doing what it should and triggers an easter egg!
			# I discovered this because I am super clever and not at all because I misread the spec
			self.writebyte(index, value)
		elif (curr == 11): # mod
			index = self.readnext('raw')
			value = (self.readnext() % self.readnext()) % 32768
			self.writebyte(index, value)
		elif (curr == 12): # and
			index = self.readnext('raw')
			self.writebyte(index, self.readnext() & self.readnext() % 32768)
		elif (curr == 13): # or
			index = self.readnext('raw')
			self.writebyte(index, self.readnext() | self.readnext() % 32768)
		elif (curr == 14): # not
			index = self.readnext('raw')
			value = self.readnext()
			notval = (~value & 0xFFFF) % 32768 # mgeneral gave me this
			self.debug('bitwise not of ' + str(value) + ' is ' + str(notval))
			self.writebyte(index, notval)
		elif (curr == 15): # rmem
			index = self.readnext(False)
			tmp = self.index + 1
			self.setindex(self.readnext())
			value = self.readbyte()
			self.setindex(tmp)
			self.debug('rmem with register ' + str(index) + ' and value ' + str(value))
			self.setregister(index, value)
		elif (curr == 16): # wmem
			index = self.readnext()
			value = self.readnext()
			self.debug('wmem with index ' + str(index) + ' and value ' + str(value))
			self.writebyte(index, value)
		elif (curr == 17): # call
			self.call()
		elif (curr == 18): # ret
			index = self.stack.pop()
			self.debug('ret returned ' + str(index))
			self.jmp(index)
		elif (curr == 19): # out
			if (self.isdebug() == False):
				print chr(self.readnext()),
			else:
				self.setindex(self.index + 1)
		elif (curr == 21): # noop
			pass
		else:
			sys.exit(str(curr) + ' is not configured')
		self.setindex(self.index + 1)
		self.parse()

challenge = SynChallenge()
challenge.parse()
