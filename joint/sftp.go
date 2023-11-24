package joint

import (
	"io/fs"
	"net/url"
	"path"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

type SftpFileStat = sftp.FileStat

// SftpPwd return SFTP current directory. It's used cache to avoid
// extra calls to SFTP-server to get current directory for every call.
func SftpPwd(ftpaddr string, client *sftp.Client) (pwd string) {
	pwdmux.RLock()
	pwd, ok := pwdmap[ftpaddr]
	pwdmux.RUnlock()
	if !ok {
		var err error
		if pwd, err = client.Getwd(); err == nil {
			pwdmux.Lock()
			pwdmap[ftpaddr] = pwd
			pwdmux.Unlock()
		}
	}
	return
}

// SftpJoint create SSH-connection to SFTP-server, login with provided by
// given URL credentials, and gets a once current directory.
// Joint can be used for followed files access.
type SftpJoint struct {
	key    string // address of SFTP-service, i.e. sftp://user:pass@example.com
	conn   *ssh.Client
	client *sftp.Client
	pwd    string

	path string // path inside of SFTP-service without PWD
	*sftp.File
}

func (jnt *SftpJoint) Make(urladdr string) (err error) {
	jnt.key = urladdr
	var u *url.URL
	if u, err = url.Parse(urladdr); err != nil {
		return
	}
	var pass, _ = u.User.Password()
	var config = &ssh.ClientConfig{
		User: u.User.Username(),
		Auth: []ssh.AuthMethod{
			ssh.Password(pass),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	if jnt.conn, err = ssh.Dial("tcp", u.Host, config); err != nil {
		return
	}
	if jnt.client, err = sftp.NewClient(jnt.conn); err != nil {
		return
	}
	jnt.pwd = SftpPwd(u.Host, jnt.client)
	return
}

func (jnt *SftpJoint) Cleanup() (err error) {
	err = jnt.client.Close()
	if err1 := jnt.conn.Close(); err1 != nil {
		err = err1
	}
	return
}

func (jnt *SftpJoint) Key() string {
	return jnt.key
}

func (jnt *SftpJoint) Busy() bool {
	return jnt.File != nil
}

// Opens new connection for any some one file with given full SFTP URL.
func (jnt *SftpJoint) Open(fpath string) (file fs.File, err error) {
	jnt.path = fpath
	if jnt.File, err = jnt.client.Open(path.Join(jnt.pwd, fpath)); err != nil {
		return
	}
	file = jnt
	return
}

func (jnt *SftpJoint) Close() (err error) {
	err = jnt.File.Close()
	jnt.path = ""
	jnt.File = nil
	PutJoint(jnt)
	return
}

func (jnt *SftpJoint) Info(fpath string) (fs.FileInfo, error) {
	return jnt.client.Stat(path.Join(jnt.pwd, fpath))
}

func (jnt *SftpJoint) ReadDir(fpath string) ([]fs.FileInfo, error) {
	return jnt.client.ReadDir(path.Join(jnt.pwd, fpath))
}

// The End.
