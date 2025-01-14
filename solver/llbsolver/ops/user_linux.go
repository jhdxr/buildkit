package ops

import (
	"os"
	"syscall"

	"github.com/containerd/continuity/fs"
	"github.com/moby/buildkit/snapshot"
	"github.com/moby/buildkit/solver/pb"
	"github.com/moby/buildkit/worker"
	"github.com/opencontainers/runc/libcontainer/user"
	"github.com/pkg/errors"
	copy "github.com/tonistiigi/fsutil/copy"
)

func getReadUserFn(worker worker.Worker) func(chopt *pb.ChownOpt, mu, mg snapshot.Mountable) (*copy.User, error) {
	return readUser
}

func readUser(chopt *pb.ChownOpt, mu, mg snapshot.Mountable) (*copy.User, error) {
	if chopt == nil {
		return nil, nil
	}
	var us copy.User
	if chopt.User != nil {
		switch u := chopt.User.User.(type) {
		case *pb.UserOpt_ByName:
			if mu == nil {
				return nil, errors.Errorf("invalid missing user mount")
			}

			lm := snapshot.LocalMounter(mu)
			dir, err := lm.Mount()
			if err != nil {
				return nil, err
			}
			defer lm.Unmount()

			passwdPath, err := user.GetPasswdPath()
			if err != nil {
				return nil, err
			}

			passwdPath, err = fs.RootPath(dir, passwdPath)
			if err != nil {
				return nil, err
			}

			ufile, err := os.Open(passwdPath)
			if errors.Is(err, os.ErrNotExist) || errors.Is(err, syscall.ENOTDIR) {
				// Couldn't open the file. Considering this case as not finding the user in the file.
				break
			}
			if err != nil {
				return nil, err
			}
			defer ufile.Close()

			users, err := user.ParsePasswdFilter(ufile, func(uu user.User) bool {
				return uu.Name == u.ByName.Name
			})
			if err != nil {
				return nil, err
			}

			if len(users) > 0 {
				us.UID = users[0].Uid
				us.GID = users[0].Gid
			}
		case *pb.UserOpt_ByID:
			us.UID = int(u.ByID)
			us.GID = int(u.ByID)
		}
	}

	if chopt.Group != nil {
		switch u := chopt.Group.User.(type) {
		case *pb.UserOpt_ByName:
			if mg == nil {
				return nil, errors.Errorf("invalid missing group mount")
			}

			lm := snapshot.LocalMounter(mg)
			dir, err := lm.Mount()
			if err != nil {
				return nil, err
			}
			defer lm.Unmount()

			groupPath, err := user.GetGroupPath()
			if err != nil {
				return nil, err
			}

			groupPath, err = fs.RootPath(dir, groupPath)
			if err != nil {
				return nil, err
			}

			gfile, err := os.Open(groupPath)
			if errors.Is(err, os.ErrNotExist) || errors.Is(err, syscall.ENOTDIR) {
				// Couldn't open the file. Considering this case as not finding the group in the file.
				break
			}
			if err != nil {
				return nil, err
			}
			defer gfile.Close()

			groups, err := user.ParseGroupFilter(gfile, func(g user.Group) bool {
				return g.Name == u.ByName.Name
			})
			if err != nil {
				return nil, err
			}

			if len(groups) > 0 {
				us.GID = groups[0].Gid
			}
		case *pb.UserOpt_ByID:
			us.GID = int(u.ByID)
		}
	}

	return &us, nil
}
