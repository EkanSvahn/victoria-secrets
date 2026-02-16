import { createRouter, createWebHistory } from 'vue-router'
import CreateView from '../views/CreateView.vue'
import InfoView from '../views/InfoView.vue'
import OpenView from '../views/OpenView.vue'

export const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', redirect: '/secret' },
    { path: '/secret', component: CreateView },
    { path: '/info', component: InfoView },
    { path: '/s/:id', component: OpenView, props: true }
  ]
})
