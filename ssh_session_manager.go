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
	sessionCache  map[string]*SSHSession
	sessionLocker map[string]*sync.Mutex
	globalLocker  *sync.Mutex
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
	sessionManager.globalLocker = new(sync.Mutex)
	//启动自动清理的线程，清理10分钟未使用的session缓存
	sessionManager.RunAutoClean()
	return sessionManager
}

/**
 * 给指定的session上锁
 * @param  sessionKey:session的索引键值
 * @author shenbowei
 */
func (this *SessionManager) LockSession(sessionKey string) {
	mutex, ok := this.sessionLocker[sessionKey]
	if !ok {
		//如果获取不到锁，需要创建锁，主要更新锁存的时候需要上全局锁
		mutex = new(sync.Mutex)
		this.globalLocker.Lock()
		this.sessionLocker[sessionKey] = mutex
		this.globalLocker.Unlock()
	}
	this.sessionLocker[sessionKey].Lock()
}

/**
 * 给指定的session解锁
 * @param  sessionKey:session的索引键值
 * @author shenbowei
 */
func (this *SessionManager) UnlockSession(sessionKey string) {
	this.sessionLocker[sessionKey].Unlock()
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
		return err
	}
	//初始化session，包括等待登录输出和禁用分页
	this.initSession(mySession, brand)
	//更新session的缓存
	this.sessionCache[sessionKey] = mySession
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
	session.ReadChannelExpect(1, "#", ">", "]")
}

/**
 * 从缓存中获取session。如果不存在或者不可用，则重新创建
 * @param  user ssh连接的用户名, password 密码, ipPort 交换机的ip和端口
 * @return SSHSession
 * @author shenbowei
 */
func (this *SessionManager) GetSession(user, password, ipPort, brand string) (*SSHSession, error) {
	sessionKey := user + "_" + password + "_" + ipPort
	session, ok := this.sessionCache[sessionKey]
	if ok {
		//返回前要验证是否可用，不可用要重新创建并更新缓存
		if session.CheckSelf() {
			session.UpdateLastUseTime()
			return session, nil
		}
	}
	//如果不存在或者验证失败，需要重新连接，并更新缓存
	if err := this.updateSession(user, password, ipPort, brand); err != nil {
		return nil, err
	} else {
		return this.sessionCache[sessionKey], nil
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
			this.globalLocker.Lock()
			for _, sessionKey := range timeoutSessionIndex {
				delete(this.sessionCache, sessionKey)
			}
			this.globalLocker.Unlock()
			time.Sleep(time.Second)
		}
	}()
}

/**
 * 获取所有超时（10分钟未使用）session在cache的sessionKey
 * @return []string 所有超时的sessionKey数组
 * @author shenbowei
 */
func (this *SessionManager) getTimeoutSessionIndex() []string {
	defer func() {
		if err := recover(); err != nil {
			LogError("SSHSessionManager getTimeoutSessionIndex err:%s", err)
		}
	}()
	timeoutSessionIndex := make([]string, 0)
	for sessionKey, SSHSession := range this.sessionCache {
		timeDuratime := time.Now().Sub(SSHSession.GetLastUseTime())
		if timeDuratime.Minutes() > 10 {
			SSHSession.Close()
			timeoutSessionIndex = append(timeoutSessionIndex, sessionKey)
		}
	}
	return timeoutSessionIndex
}