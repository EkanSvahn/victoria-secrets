import { createRouter, createWebHistory } from 'vue-router'
import CreateView from '../views/CreateView.vue'
import OpenView from '../views/OpenView.vue'

export const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', component: CreateView },
    { path: '/s/:id', component: OpenView, props: true }
  ]
})
