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

func (self *ServiceManager) GetService(serviceId string) Service {
	self.mutex.Lock()
	defer self.mutex.Unlock()
	for _, service := range self.services {
		if service.GetServiceId() == serviceId {
			return service
		}
	}
	return nil
}

func (self *ServiceManager) GetStartedServices() []Service {
	startedServices := make([]Service, 0)
	self.mutex.Lock()
	defer self.mutex.Unlock()
	for _, service := range self.services {
		if service.IsStarted() {
			startedServices = append(startedServices, service)
		}
	}
	return startedServices
}

func (self *ServiceManager) GetAllServices() []Service {
	newServices := make([]Service, len(self.services))
	copy(newServices, self.services)
	return newServices
}
