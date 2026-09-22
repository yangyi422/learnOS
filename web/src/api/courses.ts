import { request } from './http'
import type { Course } from '@/types/course'

interface CourseListResponse {
  data: Course[]
}

export async function listCourses(): Promise<Course[]> {
  const response = await request<CourseListResponse>('/api/v1/courses')
  return response.data
}

export async function deleteCourse(courseID: number): Promise<void> {
  await request<{ data: { deleted: boolean } }>(`/api/v1/courses/${courseID}`, { method: 'DELETE' })
}
