import { baseApi } from '../../shared/api/baseApi'
import type {
  Board,
  BoardDetail,
  Label,
  LabelInput,
  Project,
  ProjectPatch,
  Role,
  User,
  UserPage,
} from '../../shared/api/types'

export const projectApi = baseApi.injectEndpoints({
  endpoints: (build) => ({
    getProject: build.query<Project, string>({
      query: (id) => `/projects/${id}`,
      providesTags: (_r, _e, id) => [{ type: 'Project', id }],
    }),
    updateProject: build.mutation<Project, { id: string; patch: ProjectPatch }>({
      query: ({ id, patch }) => ({ url: `/projects/${id}`, method: 'PATCH', body: patch }),
      invalidatesTags: (_r, _e, { id }) => ['Project', { type: 'Project', id }],
    }),
    listMembers: build.query<User[], string>({
      query: (projectId) => `/projects/${projectId}/members`,
      providesTags: (_r, _e, projectId) => [{ type: 'Member', id: projectId }],
    }),
    addMember: build.mutation<void, { projectId: string; userId: string }>({
      query: ({ projectId, userId }) => ({
        url: `/projects/${projectId}/members`,
        method: 'POST',
        body: { userId },
      }),
      invalidatesTags: (_r, _e, { projectId }) => [{ type: 'Member', id: projectId }],
    }),
    removeMember: build.mutation<void, { projectId: string; userId: string }>({
      query: ({ projectId, userId }) => ({
        url: `/projects/${projectId}/members/${userId}`,
        method: 'DELETE',
      }),
      invalidatesTags: (_r, _e, { projectId }) => [{ type: 'Member', id: projectId }, 'Cards'],
    }),
    listLabels: build.query<Label[], string>({
      query: (projectId) => `/projects/${projectId}/labels`,
      providesTags: (_r, _e, projectId) => [{ type: 'Label', id: projectId }],
    }),
    createLabel: build.mutation<Label, { projectId: string; input: LabelInput }>({
      query: ({ projectId, input }) => ({
        url: `/projects/${projectId}/labels`,
        method: 'POST',
        body: input,
      }),
      invalidatesTags: (_r, _e, { projectId }) => [{ type: 'Label', id: projectId }],
    }),
    deleteLabel: build.mutation<void, { projectId: string; labelId: string }>({
      query: ({ labelId }) => ({ url: `/labels/${labelId}`, method: 'DELETE' }),
      invalidatesTags: (_r, _e, { projectId }) => [{ type: 'Label', id: projectId }, 'Cards'],
    }),
    listBoards: build.query<Board[], string>({
      query: (projectId) => `/projects/${projectId}/boards`,
      providesTags: (_r, _e, projectId) => [{ type: 'Board', id: `project-${projectId}` }],
    }),
    createBoard: build.mutation<BoardDetail, { projectId: string; name: string }>({
      query: ({ projectId, name }) => ({
        url: `/projects/${projectId}/boards`,
        method: 'POST',
        body: { name },
      }),
      invalidatesTags: (_r, _e, { projectId }) => [{ type: 'Board', id: `project-${projectId}` }],
    }),
    deleteBoard: build.mutation<void, { projectId: string; boardId: string }>({
      query: ({ boardId }) => ({ url: `/boards/${boardId}`, method: 'DELETE' }),
      invalidatesTags: (_r, _e, { projectId }) => [{ type: 'Board', id: `project-${projectId}` }],
    }),
    searchUsers: build.query<UserPage, string>({
      query: (q) => ({ url: '/users', params: { q, limit: 20 } }),
      providesTags: ['User'],
    }),
    listAllUsers: build.query<UserPage, { q: string; offset: number }>({
      query: ({ q, offset }) => ({ url: '/users', params: { q, limit: 50, offset } }),
      providesTags: ['User'],
    }),
    setUserRole: build.mutation<User, { userId: string; role: Role }>({
      query: ({ userId, role }) => ({
        url: `/users/${userId}/role`,
        method: 'PUT',
        body: { role },
      }),
      invalidatesTags: ['User', 'Member'],
    }),
  }),
})

export const {
  useGetProjectQuery,
  useUpdateProjectMutation,
  useListMembersQuery,
  useAddMemberMutation,
  useRemoveMemberMutation,
  useListLabelsQuery,
  useCreateLabelMutation,
  useDeleteLabelMutation,
  useListBoardsQuery,
  useCreateBoardMutation,
  useDeleteBoardMutation,
  useSearchUsersQuery,
  useListAllUsersQuery,
  useSetUserRoleMutation,
} = projectApi
