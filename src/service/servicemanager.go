package service

import (
	"sync"
)

type ServiceManager struct {
	services []Service
	mutex    sync.Mutex
}

func (self *ServiceManager) AddService(service Service) {
	self.mutex.Lock()
	defer self.mutex.Unlock()
	self.services = append(self.services, service)
}

func (self *ServiceManager) RemoveService(service Service) {
	self.mutex.Lock()
	defer self.mutex.Unlock()
	for i, oldServices := range self.services {
		if oldServices == service {
			self.services = append(self.services[0:i], self.services[i+1:]...)
			break
		}
	}
}

func (self *ServiceManager) GetAllServices() []Service {
	self.mutex.Lock()
	defer self.mutex.Unlock()
	newServices := make([]Service, len(self.services))
	copy(newServices, self.services)
	return newServices
}
