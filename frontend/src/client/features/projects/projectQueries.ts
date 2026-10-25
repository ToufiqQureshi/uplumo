import { queryOptions, useQuery } from "@tanstack/react-query";
import { getProjects, type ProjectMarket } from "./projectApi";

// The org's active projects. Invalidating this key also refreshes the
// archived list, which lives under ["projects", "archived"].
export const projectsQueryOptions = () =>
  queryOptions({
    queryKey: ["projects"],
    queryFn: getProjects,
  });

/** The project's default market, or undefined until the projects query resolves. */
export function useProjectMarket(projectId: string): ProjectMarket | undefined {
  const projectsQuery = useQuery(projectsQueryOptions());
  return projectsQuery.data?.find((project) => project.id === projectId);
}
