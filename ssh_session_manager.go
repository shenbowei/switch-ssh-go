package ssh

import (
	"sync"
	"time"
)

var (
	HuaweiNoPage = "screen-length 0 temporary"
	H3cNoPage    = "screen-length disable"
	CiscoNoPage  = "terminal length 0"
)

var sessionManager = NewSessionManager()

/**
 * session（SSHSession）的管理类，会统一缓存打开的session，自动处理未使用超过10分钟的session
 * @attr sessionCache:缓存所有打开的map（10分钟内使用过的），sessionLocker设备锁，globalLocker全局锁
 * @author shenbowei
 */
type SessionManager struct {
	sessionCache           map[string]*SSHSession
	sessionLocker          map[string]*sync.Mutex
	sessionCacheLocker     *sync.RWMutex
	sessionLockerMapLocker *sync.RWMutex
}

/**
 * 创建一个SessionManager，相当于SessionManager的构造函数
 * @return SessionManager实例
 * @author shenbowei
 */
func NewSessionManager() *SessionManager {
	sessionManager := new(SessionManager)
	sessionManager.sessionCache = make(map[string]*SSHSession, 0)
	sessionManager.sessionLocker = make(map[string]*sync.Mutex, 0)
	sessionManager.sessionCacheLocker = new(sync.RWMutex)
	sessionManager.sessionLockerMapLocker = new(sync.RWMutex)
	//启动自动清理的线程，清理10分钟未使用的session缓存
	sessionManager.RunAutoClean()
	return sessionManager
}

func (this *SessionManager) SetSessionCache(sessionKey string, session *SSHSession) {
	this.sessionCacheLocker.Lock()
	defer this.sessionCacheLocker.Unlock()
	this.sessionCache[sessionKey] = session
}

func (this *SessionManager) GetSessionCache(sessionKey string) *SSHSession {
	this.sessionCacheLocker.RLock()
	defer this.sessionCacheLocker.RUnlock()
	cache, ok := this.sessionCache[sessionKey]
	if ok {
		return cache
	} else {
		return nil
	}
}

/**
 * 给指定的session上锁
 * @param  sessionKey:session的索引键值
 * @author shenbowei
 */
func (this *SessionManager) LockSession(sessionKey string) {
	this.sessionLockerMapLocker.RLock()
	mutex, ok := this.sessionLocker[sessionKey]
	this.sessionLockerMapLocker.RUnlock()
	if !ok {
		//如果获取不到锁，需要创建锁，主要更新锁存的时候需要上全局锁
		mutex = new(sync.Mutex)
		this.sessionLockerMapLocker.Lock()
		this.sessionLocker[sessionKey] = mutex
		this.sessionLockerMapLocker.Unlock()
	}
	mutex.Lock()
}

/**
 * 给指定的session解锁
 * @param  sessionKey:session的索引键值
 * @author shenbowei
 */
func (this *SessionManager) UnlockSession(sessionKey string) {
	this.sessionLockerMapLocker.RLock()
	this.sessionLocker[sessionKey].Unlock()
	this.sessionLockerMapLocker.RUnlock()
}

/**
 * 更新session缓存中的session，连接设备，打开会话，初始化会话（等待登录，识别设备类型，执行禁止分页），添加到缓存
 * @param  user ssh连接的用户名, password 密码, ipPort 交换机的ip和端口
 * @return 执行的错误
 * @author shenbowei
 */
func (this *SessionManager) updateSession(user, password, ipPort, brand string) error {
	sessionKey := user + "_" + password + "_" + ipPort
	mySession, err := NewSSHSession(user, password, ipPort)
	if err != nil {
		LogError("NewSSHSession err:%s", err.Error())
		return err
	}
	//初始化session，包括等待登录输出和禁用分页
	this.initSession(mySession, brand)
	//更新session的缓存
	this.SetSessionCache(sessionKey, mySession)
	return nil
}

/**
 * 初始化会话（等待登录，识别设备类型，执行禁止分页）
 * @param  session:需要执行初始化操作的SSHSession
 * @author shenbowei
 */
func (this *SessionManager) initSession(session *SSHSession, brand string) {
	if brand != HUAWEI && brand != H3C && brand != CISCO {
		//如果传入的设备型号不匹配则自己获取
		brand = session.GetSSHBrand()
	}
	switch brand {
	case HUAWEI:
		session.WriteChannel(HuaweiNoPage)
		break
	case H3C:
		session.WriteChannel(H3cNoPage)
		break
	case CISCO:
		session.WriteChannel(CiscoNoPage)
		break
	default:
		return
	}
	session.ReadChannelExpect(time.Second, "#", ">", "]")
}

/**
 * 从缓存中获取session。如果不存在或者不可用，则重新创建
 * @param  user ssh连接的用户名, password 密码, ipPort 交换机的ip和端口
 * @return SSHSession
 * @author shenbowei
 */
func (this *SessionManager) GetSession(user, password, ipPort, brand string) (*SSHSession, error) {
	sessionKey := user + "_" + password + "_" + ipPort
	session := this.GetSessionCache(sessionKey)
	if session != nil {
		//返回前要验证是否可用，不可用要重新创建并更新缓存
		if session.CheckSelf() {
			LogDebug("-----GetSession from cache-----")
			session.UpdateLastUseTime()
			return session, nil
		}
		LogDebug("Check session failed")
	}
	//如果不存在或者验证失败，需要重新连接，并更新缓存
	if err := this.updateSession(user, password, ipPort, brand); err != nil {
		LogError("SSH session pool updateSession err:%s", err.Error())
		return nil, err
	} else {
		return this.GetSessionCache(sessionKey), nil
	}
}

/**
 * 开始自动清理session缓存中未使用超过10分钟的session
 * @author shenbowei
 */
func (this *SessionManager) RunAutoClean() {
	go func() {
		for {
			timeoutSessionIndex := this.getTimeoutSessionIndex()
			this.sessionCacheLocker.Lock()
			for _, sessionKey := range timeoutSessionIndex {
				this.LockSession(sessionKey)
				delete(this.sessionCache, sessionKey)
				this.UnlockSession(sessionKey)
			}
			this.sessionCacheLocker.Unlock()
			time.Sleep(30 * time.Second)
		}
	}()
}

/**
 * 获取所有超时（10分钟未使用）session在cache的sessionKey
 * @return []string 所有超时的sessionKey数组
 * @author shenbowei
 */
func (this *SessionManager) getTimeoutSessionIndex() []string {
	timeoutSessionIndex := make([]string, 0)
	this.sessionCacheLocker.RLock()
	defer func() {
		this.sessionCacheLocker.RUnlock()
		if err := recover(); err != nil {
			LogError("SSHSessionManager getTimeoutSessionIndex err:%s", err)
		}
	}()
	for sessionKey, SSHSession := range this.sessionCache {
		timeDuratime := time.Now().Sub(SSHSession.GetLastUseTime())
		if timeDuratime.Minutes() > 10 {
			LogDebug("RunAutoClean close session<%s, unuse time=%s>", sessionKey, timeDuratime.String())
			SSHSession.Close()
			timeoutSessionIndex = append(timeoutSessionIndex, sessionKey)
		}
	}
	return timeoutSessionIndex
}
