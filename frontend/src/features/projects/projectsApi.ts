import { baseApi } from '../../shared/api/baseApi'
import type { Project, ProjectInput } from '../../shared/api/types'

export const projectsApi = baseApi.injectEndpoints({
  endpoints: (build) => ({
    listProjects: build.query<Project[], void>({
      query: () => '/projects',
      providesTags: ['Project'],
    }),
    createProject: build.mutation<Project, ProjectInput>({
      query: (body) => ({ url: '/projects', method: 'POST', body }),
      invalidatesTags: ['Project'],
    }),
    deleteProject: build.mutation<void, string>({
      query: (id) => ({ url: `/projects/${id}`, method: 'DELETE' }),
      invalidatesTags: ['Project'],
    }),
  }),
})

export const { useListProjectsQuery, useCreateProjectMutation, useDeleteProjectMutation } =
  projectsApi
